// Package recorder contains the message-handling logic of the
// havoc-recorder binary. The Process function is dependency-injected so
// the pipeline can be unit-tested without Kafka, Postgres, or Redis.
package recorder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

// ExperimentReader looks up an experiment by id.
type ExperimentReader interface {
	GetExperiment(ctx context.Context, id domain.ID) (*domain.Experiment, error)
}

// ResultWriter persists a result. Implementations must be idempotent on
// the result's id (returns false if the row already exists).
type ResultWriter interface {
	InsertResult(ctx context.Context, r domain.ExperimentResult) (bool, error)
}

// StatusWriter updates an experiment's lifecycle status.
type StatusWriter interface {
	UpdateStatus(ctx context.Context, id domain.ID, status domain.Status, reason string) error
}

// LockReleaser releases the per-service active-experiment lock if and
// only if the caller's experiment id still owns it.
type LockReleaser interface {
	ReleaseLockIfOwner(ctx context.Context, service, experimentID string) (bool, error)
}

// Recorder bundles the dependencies needed to process a single
// havoc.results message.
type Recorder struct {
	Experiments ExperimentReader
	Results     ResultWriter
	Statuses    StatusWriter
	Locks       LockReleaser
	Logger      *slog.Logger
}

// Process handles a single havoc.results payload. The pipeline is:
//
//  1. Decode the JSON message.
//  2. Insert the result row (idempotent on id).
//  3. Look up the parent experiment.
//  4. Transition the experiment's status to its terminal state.
//  5. Release the per-service lock if this experiment still owns it.
//  6. Emit a structured log line for ELK.
//
// Steps 2–5 are each individually idempotent so duplicate Kafka
// deliveries are safe.
func (r *Recorder) Process(ctx context.Context, payload []byte) error {
	var res domain.ExperimentResult
	if err := json.Unmarshal(payload, &res); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}
	if res.ID == "" || res.ExperimentID == "" {
		return errors.New("result missing id or experiment_id")
	}

	inserted, err := r.Results.InsertResult(ctx, res)
	if err != nil {
		return fmt.Errorf("insert result: %w", err)
	}

	exp, err := r.Experiments.GetExperiment(ctx, res.ExperimentID)
	if err != nil {
		return fmt.Errorf("lookup experiment: %w", err)
	}
	if exp == nil {
		// The result is preserved but no experiment exists to update.
		// Most likely a misconfigured agent or a topic shared with
		// another deployment.
		r.Logger.Warn("result for unknown experiment",
			"experiment_id", res.ExperimentID, "result_id", res.ID)
		return nil
	}

	final := outcomeToStatus(res.Outcome)
	if err := r.Statuses.UpdateStatus(ctx, exp.ID, final, res.ErrorMessage); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	released, err := r.Locks.ReleaseLockIfOwner(ctx, exp.ServiceName(), string(exp.ID))
	if err != nil {
		// Lock release is best-effort: it carries a TTL so it cannot
		// strand. Log and continue rather than NACKing the message.
		r.Logger.Warn("release lock failed",
			"experiment_id", exp.ID, "service", exp.ServiceName(), "err", err)
	}

	r.Logger.Info("result recorded",
		"experiment_id", exp.ID,
		"result_id", res.ID,
		"outcome", res.Outcome,
		"status", final,
		"agent_node", res.AgentNode,
		"affected_pods", res.AffectedPods,
		"duration_ms", res.CompletedAt.Sub(res.StartedAt).Milliseconds(),
		"duplicate", !inserted,
		"lock_released", released,
	)
	return nil
}

// outcomeToStatus collapses an agent outcome into the experiment's
// terminal lifecycle status. Failures are still terminal — the
// experiment ran to completion, just unsuccessfully.
func outcomeToStatus(o domain.Outcome) domain.Status {
	if o == domain.OutcomeAborted {
		return domain.StatusAborted
	}
	return domain.StatusCompleted
}
