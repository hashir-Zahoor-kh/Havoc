// Package chaos contains the actual fault-injection implementations the
// agent runs against target pods. Each action is a small struct that
// satisfies the Action interface, which keeps the agent's runner free of
// any per-action conditionals.
package chaos

import (
	"context"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

// Action runs a single chaos action against a single pod. Implementations
// must respect ctx cancellation — the agent cancels ctx when a kill
// switch is engaged or an abort message arrives, and Execute should
// return promptly so the agent can publish an aborted result.
type Action interface {
	// Type is the action type this implementation handles.
	Type() domain.ActionType
	// Execute applies the action to the named pod in the experiment's
	// namespace. The error is wrapped into an OutcomeFailure result; a
	// ctx-cancelled return is mapped to OutcomeAborted.
	Execute(ctx context.Context, exp domain.Experiment, podName string) error
}

// Registry indexes Actions by their domain.ActionType.
type Registry map[domain.ActionType]Action

// NewRegistry builds a registry from a list of actions. Later entries
// override earlier ones, which is convenient for tests.
func NewRegistry(actions ...Action) Registry {
	r := Registry{}
	for _, a := range actions {
		r[a.Type()] = a
	}
	return r
}
