// Package domain defines the core types that flow between the control plane,
// agents, and recorder. Everything here is plain data — no I/O, no logging,
// no dependencies on Kafka/Postgres/Redis. Validation lives alongside the
// types; guardrail logic that needs external state lives in internal/safety.
package domain

import "time"

// ID is a string-encoded UUID used for experiments and results.
// Kept as a string alias so the domain package has no external dependencies;
// ID generation lives in the control plane.
type ID string

// ActionType identifies one of the supported chaos actions.
type ActionType string

const (
	ActionPodKill        ActionType = "pod_kill"
	ActionNetworkLatency ActionType = "network_latency"
	ActionCPUPressure    ActionType = "cpu_pressure"
)

// Valid reports whether the action type is one of the supported values.
func (a ActionType) Valid() bool {
	switch a {
	case ActionPodKill, ActionNetworkLatency, ActionCPUPressure:
		return true
	}
	return false
}

// Status is the lifecycle state of an experiment.
type Status string

const (
	StatusScheduled Status = "scheduled"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusAborted   Status = "aborted"
	StatusRejected  Status = "rejected"
)

// Outcome is the terminal state of an agent-executed action.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeAborted Outcome = "aborted"
)

// Parameters holds action-specific tuning knobs. Only the fields relevant
// to the experiment's ActionType are populated; the rest are zero.
type Parameters struct {
	LatencyMilliseconds int `json:"latency_ms,omitempty"`
	CPUPercent          int `json:"cpu_percent,omitempty"`
}

// Experiment is the request-time record of a chaos experiment.
//
// TargetPods is filled in by the control plane *after* the guardrails
// pass. It names the specific pods this experiment is permitted to
// affect — every agent self-filters by intersecting TargetPods with
// the matching pods on its own node, which is how Havoc enforces blast
// radius across a DaemonSet of independent agents.
type Experiment struct {
	ID              ID                `json:"id"`
	CreatedAt       time.Time         `json:"created_at"`
	ScheduledFor    time.Time         `json:"scheduled_for"`
	ActionType      ActionType        `json:"action_type"`
	TargetSelector  map[string]string `json:"target_selector"`
	TargetNamespace string            `json:"target_namespace"`
	TargetPods      []string          `json:"target_pods,omitempty"`
	DurationSeconds int               `json:"duration_seconds"`
	Parameters      Parameters        `json:"parameters"`
	Status          Status            `json:"status"`
	RejectionReason string            `json:"rejection_reason,omitempty"`
}

// ServiceName is the human-readable identifier used for the Redis
// active-experiment lock. It prefers "namespace/app" and falls back to
// "namespace" when no app label is set.
func (e Experiment) ServiceName() string {
	if v := e.TargetSelector["app"]; v != "" {
		return e.TargetNamespace + "/" + v
	}
	return e.TargetNamespace
}

// ExperimentResult is the agent's report of what it actually did.
type ExperimentResult struct {
	ID           ID        `json:"id"`
	ExperimentID ID        `json:"experiment_id"`
	AgentNode    string    `json:"agent_node"`
	AffectedPods []string  `json:"affected_pods"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	Outcome      Outcome   `json:"outcome"`
	ErrorMessage string    `json:"error_message,omitempty"`
}
