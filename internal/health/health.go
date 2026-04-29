// Package health exposes a tiny HTTP server that surfaces liveness and
// readiness probes for the agent and recorder binaries. The control
// plane already has its own HTTP server (see cmd/havoc-control/server.go)
// and serves /healthz from there directly — only the two pure-consumer
// binaries need this helper.
//
// /healthz returns 200 unconditionally as long as the process is up.
// Kubernetes uses it as the liveness probe — failing it triggers a
// restart, so the bar is "the process exists and the goroutine
// scheduler is responsive."
//
// /readyz returns 200 only after MarkReady() is called. The agent and
// recorder mark themselves ready once all dependencies (Kafka,
// Postgres, Redis, the k8s client) have been dialed successfully —
// before that, readiness is 503 and Kubernetes won't include the pod
// in any service endpoints. That keeps a half-initialized recorder
// out of the result-consumer rotation.
package health

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"
)

// Server is the readiness/liveness HTTP server.
type Server struct {
	srv   *http.Server
	ready atomic.Bool
}

// New constructs a Server that will listen on addr (e.g. ":8081").
// Call Start in a goroutine and MarkReady once dependencies are up.
func New(addr string) *Server {
	s := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	s.srv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// MarkReady flips /readyz from 503 to 200. Idempotent.
func (s *Server) MarkReady() { s.ready.Store(true) }

// MarkNotReady flips /readyz back to 503. Useful during graceful
// shutdown so Kubernetes drains traffic before the process exits.
func (s *Server) MarkNotReady() { s.ready.Store(false) }

// IsReady reports the current readiness state. Exposed for tests.
func (s *Server) IsReady() bool { return s.ready.Load() }

// Start runs the HTTP server until ctx is canceled or the listener
// fails. It always returns either the listener error or the result
// of Shutdown — never io.EOF or a wrapped context.Canceled, so the
// caller can decide whether to log it.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Bounded shutdown so a hung connection can't block pod
		// termination past the kubelet's grace period.
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return s.srv.Shutdown(shutCtx)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
