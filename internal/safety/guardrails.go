// Package safety enforces Havoc's scheduling guardrails: global kill
// switch, per-service active-experiment locking, blast radius limits, and
// blackout windows. The logic is stateless; external state (Redis,
// Postgres, Kubernetes) is supplied through the interfaces below so the
// evaluator can be unit-tested without any infrastructure.
package safety

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

// Sentinel guardrail errors. The control plane maps these to HTTP 409
// rejection responses; each carries enough context for operators to
// understand why the experiment was refused.
var (
	ErrKillSwitchEngaged   = errors.New("global kill switch is engaged")
	ErrActiveExperiment    = errors.New("another experiment is already active on this service")
	ErrNoMatchingPods      = errors.New("target selector matches no pods")
	ErrBlastRadiusExceeded = errors.New("experiment would exceed blast radius limit")
	ErrInBlackoutWindow    = errors.New("target time falls within a blackout window")
)

// KillSwitch reports whether the global kill switch is currently engaged.
type KillSwitch interface {
	KillSwitchEngaged(ctx context.Context) (bool, error)
}

// LockStore checks whether a given service already has an in-flight
// experiment.
type LockStore interface {
	IsLocked(ctx context.Context, service string) (bool, error)
}

// PodLister resolves a label selector to a list of matching pod names.
type PodLister interface {
	ListPods(ctx context.Context, namespace string, selector map[string]string) ([]string, error)
}

// BlackoutSource returns the configured blackout windows.
type BlackoutSource interface {
	ListWindows(ctx context.Context) ([]BlackoutWindow, error)
}

// Config holds tuning knobs for the guardrail evaluator. A zero
// MaxBlastRadiusPercent is treated as 25 — the documented default.
type Config struct {
	MaxBlastRadiusPercent int
}

// Guard evaluates all four guardrails against a proposed experiment.
type Guard struct {
	KillSwitch KillSwitch
	Locks      LockStore
	Pods       PodLister
	Blackouts  BlackoutSource
	Config     Config
}

// Check returns nil if the experiment is safe to schedule, otherwise the
// first guardrail error to trip. Checks run in the order:
// kill switch → active lock → blast radius → blackout window.
func (g *Guard) Check(ctx context.Context, e domain.Experiment, now time.Time) error {
	engaged, err := g.KillSwitch.KillSwitchEngaged(ctx)
	if err != nil {
		return fmt.Errorf("kill switch check: %w", err)
	}
	if engaged {
		return ErrKillSwitchEngaged
	}

	locked, err := g.Locks.IsLocked(ctx, e.ServiceName())
	if err != nil {
		return fmt.Errorf("active lock check: %w", err)
	}
	if locked {
		return ErrActiveExperiment
	}

	pods, err := g.Pods.ListPods(ctx, e.TargetNamespace, e.TargetSelector)
	if err != nil {
		return fmt.Errorf("pod list: %w", err)
	}
	if len(pods) == 0 {
		return fmt.Errorf("%w: namespace=%q selector=%v", ErrNoMatchingPods, e.TargetNamespace, e.TargetSelector)
	}
	maxPct := g.Config.MaxBlastRadiusPercent
	if maxPct == 0 {
		maxPct = 25
	}
	// Every supported action currently affects a single pod. (100 / n) > maxPct → reject.
	affected := 1
	if 100*affected > maxPct*len(pods) {
		return fmt.Errorf("%w: %d of %d pods exceeds %d%%", ErrBlastRadiusExceeded, affected, len(pods), maxPct)
	}

	windows, err := g.Blackouts.ListWindows(ctx)
	if err != nil {
		return fmt.Errorf("blackout list: %w", err)
	}
	at := e.ScheduledFor
	if at.IsZero() {
		at = now
	}
	for _, w := range windows {
		active, err := w.Active(at)
		if err != nil {
			return fmt.Errorf("blackout %q: %w", w.Name, err)
		}
		if active {
			return fmt.Errorf("%w: %q", ErrInBlackoutWindow, w.Name)
		}
	}
	return nil
}
