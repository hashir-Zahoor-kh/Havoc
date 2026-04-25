// Package api defines the request/response types exchanged between the
// Havoc CLI and the control-plane HTTP server. The CLI and server both
// import this package so the wire schema is single-sourced.
package api

import "time"

// ScheduleRequest is the body of POST /v1/experiments.
type ScheduleRequest struct {
	Action          string            `json:"action"`
	TargetSelector  map[string]string `json:"target_selector"`
	TargetNamespace string            `json:"target_namespace"`
	DurationSeconds int               `json:"duration_seconds"`
	LatencyMS       int               `json:"latency_ms,omitempty"`
	CPUPercent      int               `json:"cpu_percent,omitempty"`
	ScheduledFor    *time.Time        `json:"scheduled_for,omitempty"`
}

// ScheduleResponse is returned when an experiment is accepted.
type ScheduleResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ExperimentView is the list/detail representation of an experiment.
type ExperimentView struct {
	ID              string            `json:"id"`
	CreatedAt       time.Time         `json:"created_at"`
	ScheduledFor    *time.Time        `json:"scheduled_for,omitempty"`
	Action          string            `json:"action"`
	TargetSelector  map[string]string `json:"target_selector"`
	TargetNamespace string            `json:"target_namespace"`
	DurationSeconds int               `json:"duration_seconds"`
	LatencyMS       int               `json:"latency_ms,omitempty"`
	CPUPercent      int               `json:"cpu_percent,omitempty"`
	Status          string            `json:"status"`
	RejectionReason string            `json:"rejection_reason,omitempty"`
}

// ListResponse is returned by GET /v1/experiments.
type ListResponse struct {
	Experiments []ExperimentView `json:"experiments"`
}

// ErrorResponse is the body of all non-2xx responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// BlackoutRequest is the body of POST /v1/blackouts.
type BlackoutRequest struct {
	Name            string `json:"name"`
	CronExpression  string `json:"cron_expression"`
	DurationMinutes int    `json:"duration_minutes"`
}

// BlackoutView is an item in the blackout listing.
type BlackoutView struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	CronExpression  string `json:"cron_expression"`
	DurationMinutes int    `json:"duration_minutes"`
}

// BlackoutListResponse is returned by GET /v1/blackouts.
type BlackoutListResponse struct {
	Blackouts []BlackoutView `json:"blackouts"`
}
