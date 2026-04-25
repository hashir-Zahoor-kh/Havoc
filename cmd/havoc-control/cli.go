package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashir-Zahoor-kh/Havoc/internal/api"
	"github.com/hashir-Zahoor-kh/Havoc/internal/config"
)

// labelSelector is a flag.Value that parses repeated key=value flags into
// a map (e.g. --target app=payments --target tier=backend).
type labelSelector map[string]string

func (l *labelSelector) String() string {
	if l == nil {
		return ""
	}
	parts := make([]string, 0, len(*l))
	for k, v := range *l {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

func (l *labelSelector) Set(s string) error {
	if *l == nil {
		*l = labelSelector{}
	}
	for _, kv := range strings.Split(s, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return fmt.Errorf("expected key=value, got %q", kv)
		}
		(*l)[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return nil
}

func runSchedule(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("schedule", flag.ContinueOnError)
	var (
		action     = fs.String("action", "", "chaos action: pod-kill | latency | cpu-pressure")
		namespace  = fs.String("namespace", "", "target namespace")
		duration   = fs.Duration("duration", 60*time.Second, "experiment duration (e.g. 60s, 5m)")
		latency    = fs.Duration("latency", 0, "injected latency for the latency action (e.g. 500ms)")
		cpuPercent = fs.Int("cpu-percent", 0, "CPU pressure percentage (1..100, default 80)")
		schedAt    = fs.String("at", "", "RFC3339 timestamp at which to schedule the experiment")
		target     labelSelector
	)
	fs.Var(&target, "target", "label selector key=value (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *action == "" || *namespace == "" || len(target) == 0 {
		return errors.New("--action, --namespace, and at least one --target are required")
	}

	req := api.ScheduleRequest{
		Action:          *action,
		TargetSelector:  target,
		TargetNamespace: *namespace,
		DurationSeconds: int((*duration).Seconds()),
		LatencyMS:       int(latency.Milliseconds()),
		CPUPercent:      *cpuPercent,
	}
	if *schedAt != "" {
		t, err := time.Parse(time.RFC3339, *schedAt)
		if err != nil {
			return fmt.Errorf("--at: %w", err)
		}
		req.ScheduledFor = &t
	}

	var resp api.ScheduleResponse
	if err := apiCall(ctx, http.MethodPost, "/v1/experiments", req, &resp); err != nil {
		return err
	}
	fmt.Printf("scheduled %s (status=%s)\n", resp.ID, resp.Status)
	return nil
}

func runList(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	limit := fs.Int("limit", 50, "maximum number of experiments to show")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var resp api.ListResponse
	if err := apiCall(ctx, http.MethodGet, fmt.Sprintf("/v1/experiments?limit=%d", *limit), nil, &resp); err != nil {
		return err
	}
	if len(resp.Experiments) == 0 {
		fmt.Println("(no experiments)")
		return nil
	}
	fmt.Printf("%-36s  %-16s  %-12s  %-20s  %s\n", "ID", "ACTION", "STATUS", "NAMESPACE", "CREATED")
	for _, e := range resp.Experiments {
		fmt.Printf("%-36s  %-16s  %-12s  %-20s  %s\n",
			e.ID, e.Action, e.Status, e.TargetNamespace,
			e.CreatedAt.Format(time.RFC3339))
	}
	return nil
}

func runStop(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	id := fs.String("id", "", "experiment id to abort")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" {
		return errors.New("--id is required")
	}
	var resp map[string]string
	if err := apiCall(ctx, http.MethodPost, fmt.Sprintf("/v1/experiments/%s/abort", url.PathEscape(*id)), nil, &resp); err != nil {
		return err
	}
	fmt.Printf("aborted %s\n", *id)
	return nil
}

func runStopAll(ctx context.Context, _ []string) error {
	var resp map[string]string
	if err := apiCall(ctx, http.MethodPost, "/v1/kill-switch/engage", nil, &resp); err != nil {
		return err
	}
	fmt.Println("kill switch engaged — all running experiments will abort")
	return nil
}

func runResume(ctx context.Context, _ []string) error {
	var resp map[string]string
	if err := apiCall(ctx, http.MethodPost, "/v1/kill-switch/disengage", nil, &resp); err != nil {
		return err
	}
	fmt.Println("kill switch disengaged")
	return nil
}

func runBlackout(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("blackout requires a subcommand: add | list | rm")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		return runBlackoutAdd(ctx, rest)
	case "list":
		return runBlackoutList(ctx, rest)
	case "rm", "remove", "delete":
		return runBlackoutRm(ctx, rest)
	}
	return fmt.Errorf("unknown blackout subcommand %q", sub)
}

func runBlackoutAdd(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("blackout add", flag.ContinueOnError)
	var (
		name     = fs.String("name", "", "blackout window name (unique)")
		cron     = fs.String("cron", "", "cron expression (5-field: min hour dom month dow)")
		duration = fs.Duration("duration", 0, "duration the window remains active each firing (e.g. 8h)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" || *cron == "" || *duration <= 0 {
		return errors.New("--name, --cron, and --duration are required")
	}
	req := api.BlackoutRequest{
		Name:            *name,
		CronExpression:  *cron,
		DurationMinutes: int(duration.Minutes()),
	}
	var resp api.BlackoutView
	if err := apiCall(ctx, http.MethodPost, "/v1/blackouts", req, &resp); err != nil {
		return err
	}
	fmt.Printf("added %s (cron=%q duration=%dm)\n", resp.Name, resp.CronExpression, resp.DurationMinutes)
	return nil
}

func runBlackoutList(ctx context.Context, _ []string) error {
	var resp api.BlackoutListResponse
	if err := apiCall(ctx, http.MethodGet, "/v1/blackouts", nil, &resp); err != nil {
		return err
	}
	if len(resp.Blackouts) == 0 {
		fmt.Println("(no blackout windows)")
		return nil
	}
	fmt.Printf("%-24s  %-20s  %s\n", "NAME", "CRON", "DURATION")
	for _, b := range resp.Blackouts {
		fmt.Printf("%-24s  %-20s  %dm\n", b.Name, b.CronExpression, b.DurationMinutes)
	}
	return nil
}

func runBlackoutRm(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: blackout rm <name>")
	}
	name := args[0]
	if err := apiCall(ctx, http.MethodDelete, "/v1/blackouts/"+url.PathEscape(name), nil, nil); err != nil {
		return err
	}
	fmt.Printf("removed %s\n", name)
	return nil
}

// apiCall is the shared HTTP helper used by every CLI subcommand.
func apiCall(ctx context.Context, method, path string, body, out any) error {
	cfg := config.LoadClient()
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}
	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, method, cfg.APIURL+path, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var er api.ErrorResponse
		if jerr := json.Unmarshal(respBody, &er); jerr == nil && er.Error != "" {
			return fmt.Errorf("api %s: %s", resp.Status, er.Error)
		}
		return fmt.Errorf("api %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
