package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashir-Zahoor-kh/Havoc/internal/api"
	"github.com/hashir-Zahoor-kh/Havoc/internal/config"
	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
	hkafka "github.com/hashir-Zahoor-kh/Havoc/internal/kafka"
	"github.com/hashir-Zahoor-kh/Havoc/internal/k8s"
	"github.com/hashir-Zahoor-kh/Havoc/internal/postgres"
	hredis "github.com/hashir-Zahoor-kh/Havoc/internal/redis"
	"github.com/hashir-Zahoor-kh/Havoc/internal/safety"
)

// blackoutAdapter bridges postgres.Store to safety.BlackoutSource without
// either package needing to know about the other.
type blackoutAdapter struct{ store *postgres.Store }

func (b blackoutAdapter) ListWindows(ctx context.Context) ([]safety.BlackoutWindow, error) {
	records, err := b.store.ListBlackouts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]safety.BlackoutWindow, 0, len(records))
	for _, r := range records {
		out = append(out, safety.BlackoutWindow{
			Name:           r.Name,
			CronExpression: r.CronExpression,
			Duration:       time.Duration(r.DurationMinutes) * time.Minute,
		})
	}
	return out, nil
}

// server wires together the dependencies the HTTP handlers need.
type server struct {
	cfg       config.Control
	logger    *slog.Logger
	store     *postgres.Store
	redis     *hredis.Client
	kafka     *hkafka.Producer
	abortsKfk *hkafka.Producer
	pods      *k8s.Client
	guard     *safety.Guard
}

func runServe(ctx context.Context, _ []string) error {
	cfg, err := config.LoadControl()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("component", "havoc-control")

	store, err := postgres.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	redis := hredis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer redis.Close()

	commands := hkafka.NewProducer(cfg.KafkaBrokers, hkafka.TopicCommands)
	defer commands.Close()

	pods, err := k8s.New(k8s.Config{InCluster: cfg.InCluster, KubeconfigPath: cfg.KubeconfigPath})
	if err != nil {
		return fmt.Errorf("kubernetes client: %w", err)
	}

	srv := &server{
		cfg:       cfg,
		logger:    logger,
		store:     store,
		redis:     redis,
		kafka:     commands,
		abortsKfk: commands,
		pods:      pods,
		guard: &safety.Guard{
			KillSwitch: redis,
			Locks:      redis,
			Pods:       pods,
			Blackouts:  blackoutAdapter{store: store},
			Config:     safety.Config{MaxBlastRadiusPercent: cfg.BlastRadiusPercent},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", srv.handleHealth)
	mux.HandleFunc("POST /v1/experiments", srv.handleSchedule)
	mux.HandleFunc("GET /v1/experiments", srv.handleList)
	mux.HandleFunc("POST /v1/experiments/{id}/abort", srv.handleStop)
	mux.HandleFunc("POST /v1/kill-switch/engage", srv.handleEngageKillSwitch)
	mux.HandleFunc("POST /v1/kill-switch/disengage", srv.handleDisengageKillSwitch)
	mux.HandleFunc("POST /v1/blackouts", srv.handleAddBlackout)
	mux.HandleFunc("GET /v1/blackouts", srv.handleListBlackouts)
	mux.HandleFunc("DELETE /v1/blackouts/{name}", srv.handleDeleteBlackout)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           logging(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Info("starting", "addr", cfg.HTTPAddr)

	errCh := make(chan error, 1)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	var req api.ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode body: %w", err))
		return
	}
	exp := domain.Experiment{
		ID:              domain.ID(uuid.NewString()),
		CreatedAt:       time.Now().UTC(),
		ActionType:      normalizeAction(req.Action),
		TargetSelector:  req.TargetSelector,
		TargetNamespace: req.TargetNamespace,
		DurationSeconds: req.DurationSeconds,
		Parameters:      domain.Parameters{LatencyMilliseconds: req.LatencyMS, CPUPercent: req.CPUPercent},
		Status:          domain.StatusScheduled,
	}
	if req.ScheduledFor != nil {
		exp.ScheduledFor = req.ScheduledFor.UTC()
	}

	if err := exp.Validate(time.Now().UTC()); err != nil {
		s.persistRejected(r.Context(), exp, err.Error())
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.guard.Check(r.Context(), exp, time.Now().UTC()); err != nil {
		s.persistRejected(r.Context(), exp, err.Error())
		writeError(w, http.StatusConflict, err)
		return
	}

	lockTTL := time.Duration(exp.DurationSeconds+s.cfg.LockBufferSeconds) * time.Second
	acquired, err := s.redis.AcquireLock(r.Context(), exp.ServiceName(), string(exp.ID), lockTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("acquire lock: %w", err))
		return
	}
	if !acquired {
		err := safety.ErrActiveExperiment
		s.persistRejected(r.Context(), exp, err.Error())
		writeError(w, http.StatusConflict, err)
		return
	}

	if err := s.store.InsertExperiment(r.Context(), exp); err != nil {
		_ = s.redis.ReleaseLock(r.Context(), exp.ServiceName())
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.kafka.Publish(r.Context(), string(exp.ID), exp); err != nil {
		// Best-effort: mark rejected and release the lock.
		_ = s.store.UpdateStatus(r.Context(), exp.ID, domain.StatusRejected, "publish failed: "+err.Error())
		_ = s.redis.ReleaseLock(r.Context(), exp.ServiceName())
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logger.Info("experiment scheduled",
		"experiment_id", exp.ID, "action", exp.ActionType,
		"namespace", exp.TargetNamespace, "service", exp.ServiceName())
	writeJSON(w, http.StatusAccepted, api.ScheduleResponse{ID: string(exp.ID), Status: string(exp.Status)})
}

func (s *server) persistRejected(ctx context.Context, exp domain.Experiment, reason string) {
	exp.Status = domain.StatusRejected
	exp.RejectionReason = reason
	if err := s.store.InsertExperiment(ctx, exp); err != nil {
		s.logger.Warn("persist rejection failed", "err", err)
	}
}

func (s *server) handleList(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := s.store.ListExperiments(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	resp := api.ListResponse{Experiments: make([]api.ExperimentView, 0, len(list))}
	for _, e := range list {
		resp.Experiments = append(resp.Experiments, experimentToView(e))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := domain.ID(r.PathValue("id"))
	exp, err := s.store.GetExperiment(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if exp == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("experiment %s not found", id))
		return
	}
	if err := s.store.UpdateStatus(r.Context(), id, domain.StatusAborted, "aborted by operator"); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := s.redis.ReleaseLockIfOwner(r.Context(), exp.ServiceName(), string(id)); err != nil {
		s.logger.Warn("release lock failed", "experiment_id", id, "err", err)
	}

	abort := map[string]any{"type": "abort", "experiment_id": string(id)}
	if err := s.abortsKfk.Publish(r.Context(), string(id), abort); err != nil {
		s.logger.Warn("publish abort failed", "experiment_id", id, "err", err)
	}
	s.logger.Info("experiment aborted", "experiment_id", id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "aborted"})
}

func (s *server) handleEngageKillSwitch(w http.ResponseWriter, r *http.Request) {
	if err := s.redis.EngageKillSwitch(r.Context(), 0); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	abort := map[string]any{"type": "kill-switch"}
	if err := s.abortsKfk.Publish(r.Context(), "killswitch", abort); err != nil {
		s.logger.Warn("publish kill-switch broadcast failed", "err", err)
	}
	s.logger.Warn("kill switch engaged")
	writeJSON(w, http.StatusOK, map[string]string{"status": "engaged"})
}

func (s *server) handleDisengageKillSwitch(w http.ResponseWriter, r *http.Request) {
	if err := s.redis.DisengageKillSwitch(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logger.Info("kill switch disengaged")
	writeJSON(w, http.StatusOK, map[string]string{"status": "disengaged"})
}

func (s *server) handleAddBlackout(w http.ResponseWriter, r *http.Request) {
	var req api.BlackoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" || req.CronExpression == "" || req.DurationMinutes <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("name, cron_expression, and duration_minutes are required"))
		return
	}
	rec := postgres.BlackoutRecord{
		ID:              domain.ID(uuid.NewString()),
		Name:            req.Name,
		CronExpression:  req.CronExpression,
		DurationMinutes: req.DurationMinutes,
	}
	if err := s.store.InsertBlackout(r.Context(), rec); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("blackout added", "name", rec.Name, "cron", rec.CronExpression, "minutes", rec.DurationMinutes)
	writeJSON(w, http.StatusCreated, api.BlackoutView{
		ID:              string(rec.ID),
		Name:            rec.Name,
		CronExpression:  rec.CronExpression,
		DurationMinutes: rec.DurationMinutes,
	})
}

func (s *server) handleListBlackouts(w http.ResponseWriter, r *http.Request) {
	records, err := s.store.ListBlackouts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	resp := api.BlackoutListResponse{Blackouts: make([]api.BlackoutView, 0, len(records))}
	for _, r := range records {
		resp.Blackouts = append(resp.Blackouts, api.BlackoutView{
			ID:              string(r.ID),
			Name:            r.Name,
			CronExpression:  r.CronExpression,
			DurationMinutes: r.DurationMinutes,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleDeleteBlackout(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ok, err := s.store.DeleteBlackout(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("blackout %q not found", name))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func experimentToView(e domain.Experiment) api.ExperimentView {
	v := api.ExperimentView{
		ID:              string(e.ID),
		CreatedAt:       e.CreatedAt,
		Action:          string(e.ActionType),
		TargetSelector:  e.TargetSelector,
		TargetNamespace: e.TargetNamespace,
		DurationSeconds: e.DurationSeconds,
		LatencyMS:       e.Parameters.LatencyMilliseconds,
		CPUPercent:      e.Parameters.CPUPercent,
		Status:          string(e.Status),
		RejectionReason: e.RejectionReason,
	}
	if !e.ScheduledFor.IsZero() {
		t := e.ScheduledFor
		v.ScheduledFor = &t
	}
	return v
}

func normalizeAction(a string) domain.ActionType {
	switch strings.ToLower(strings.ReplaceAll(a, "-", "_")) {
	case "pod_kill":
		return domain.ActionPodKill
	case "network_latency", "latency":
		return domain.ActionNetworkLatency
	case "cpu_pressure", "cpu":
		return domain.ActionCPUPressure
	}
	return domain.ActionType(a)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, api.ErrorResponse{Error: err.Error()})
}

// logging emits a structured log line for each request.
func logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(lrw, r)
		logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.status = code
	l.ResponseWriter.WriteHeader(code)
}
