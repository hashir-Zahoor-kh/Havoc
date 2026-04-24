package domain

import (
	"errors"
	"fmt"
	"time"
)

// MaxDurationSeconds caps any single experiment at one hour.
const MaxDurationSeconds = 3600

// MaxScheduleHorizon caps how far ahead an experiment may be scheduled.
const MaxScheduleHorizon = 24 * time.Hour

// DefaultCPUPercent is applied when a cpu_pressure experiment omits CPUPercent.
const DefaultCPUPercent = 80

var (
	ErrInvalidAction     = errors.New("invalid action type")
	ErrMissingSelector   = errors.New("target selector must contain at least one label")
	ErrMissingNamespace  = errors.New("target namespace is required")
	ErrInvalidDuration   = errors.New("duration must be between 1 and 3600 seconds")
	ErrInvalidSchedule   = errors.New("scheduled_for must not be more than 24 hours in the future")
	ErrInvalidLatency    = errors.New("latency_ms must be between 1 and 60000 for network_latency actions")
	ErrInvalidCPUPercent = errors.New("cpu_percent must be between 1 and 100 for cpu_pressure actions")
)

// Validate returns the first invariant the experiment violates. It checks
// shape only. Guardrails that require external state (blast radius, kill
// switch, active locks, blackout windows) live in internal/safety.
func (e Experiment) Validate(now time.Time) error {
	if !e.ActionType.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidAction, e.ActionType)
	}
	if len(e.TargetSelector) == 0 {
		return ErrMissingSelector
	}
	if e.TargetNamespace == "" {
		return ErrMissingNamespace
	}
	if e.DurationSeconds <= 0 || e.DurationSeconds > MaxDurationSeconds {
		return ErrInvalidDuration
	}
	if !e.ScheduledFor.IsZero() && e.ScheduledFor.Sub(now) > MaxScheduleHorizon {
		return ErrInvalidSchedule
	}
	switch e.ActionType {
	case ActionNetworkLatency:
		if e.Parameters.LatencyMilliseconds < 1 || e.Parameters.LatencyMilliseconds > 60000 {
			return ErrInvalidLatency
		}
	case ActionCPUPressure:
		pct := e.Parameters.CPUPercent
		if pct == 0 {
			pct = DefaultCPUPercent
		}
		if pct < 1 || pct > 100 {
			return ErrInvalidCPUPercent
		}
	}
	return nil
}
