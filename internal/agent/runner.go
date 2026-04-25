// Package agent contains the message-handling logic of the havoc-agent
// binary. The Runner consumes domain.Command envelopes from the
// havoc.commands topic, executes any pod-targeted action whose pods live
// on this agent's node, and publishes a domain.ExperimentResult for
// every attempt — including failures and aborts, since the recorder
// relies on the result to release the per-service lock.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hashir-Zahoor-kh/Havoc/internal/chaos"
	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

// PodLister returns the pods on this agent's node that match a selector.
// The agent uses it to intersect a command's pre-picked TargetPods with
// what is actually scheduled here.
type PodLister interface {
	ListPodsOnNode(ctx context.Context, node, namespace string, selector map[string]string) ([]string, error)
}

// KillSwitch reports the global kill switch state. Polled per-action and
// during long-running actions so a switch flip aborts in-flight work
// even without a Kafka broadcast.
type KillSwitch interface {
	KillSwitchEngaged(ctx context.Context) (bool, error)
}

// ResultPublisher publishes an ExperimentResult to the havoc.results
// topic. Keyed by experiment id so all results for one experiment land
// on the same partition.
type ResultPublisher interface {
	Publish(ctx context.Context, key string, v any) error
}

// Clock returns the current time. Injected so tests can pin timestamps.
type Clock func() time.Time

// IDGen returns a fresh result id. Injected so tests can stub it.
type IDGen func() domain.ID

// Runner is the per-agent dispatcher. Handle is called once per Kafka
// message; it must return promptly so the consumer can commit the
// offset. Long-running actions run in their own goroutines, tracked in
// the active map so abort and kill-switch commands can cancel them.
type Runner struct {
	NodeName  string
	Pods      PodLister
	Switch    KillSwitch
	Results   ResultPublisher
	Actions   chaos.Registry
	Logger    *slog.Logger
	Now       Clock
	NewID     IDGen
	PollEvery time.Duration

	mu     sync.Mutex
	active map[domain.ID]context.CancelFunc
}

// New constructs a Runner with sensible defaults for Now/NewID/PollEvery.
func New(nodeName string, pods PodLister, sw KillSwitch, results ResultPublisher, actions chaos.Registry, logger *slog.Logger) *Runner {
	return &Runner{
		NodeName:  nodeName,
		Pods:      pods,
		Switch:    sw,
		Results:   results,
		Actions:   actions,
		Logger:    logger,
		Now:       time.Now,
		NewID:     func() domain.ID { return domain.ID(uuid.NewString()) },
		PollEvery: 2 * time.Second,
		active:    map[domain.ID]context.CancelFunc{},
	}
}

// Handle decodes a single havoc.commands payload and dispatches on the
// envelope type. Returning nil commits the offset; returning an error
// leaves it uncommitted so the consumer reprocesses the message after a
// pod restart. Decode errors are logged and treated as nil so a single
// poison message can't wedge the consumer group.
func (r *Runner) Handle(ctx context.Context, payload []byte) error {
	var cmd domain.Command
	if err := json.Unmarshal(payload, &cmd); err != nil {
		r.Logger.Error("decode command", "err", err, "raw", string(payload))
		return nil
	}
	switch cmd.Type {
	case domain.CommandSchedule:
		if cmd.Experiment == nil {
			r.Logger.Warn("schedule missing experiment")
			return nil
		}
		r.startExperiment(ctx, *cmd.Experiment)
	case domain.CommandAbort:
		r.abort(cmd.ExperimentID)
	case domain.CommandKillSwitch:
		r.abortAll()
	default:
		r.Logger.Warn("unknown command type", "type", cmd.Type)
	}
	return nil
}

// startExperiment looks up which target pods (if any) live on this
// agent's node and, for each one, runs the action in a goroutine. The
// per-experiment cancel func is stored in active so abort/kill-switch
// can stop the work mid-flight. Self-filtering means most agents will
// see most commands but only act on a subset — enforcing blast radius
// is the control plane's job (it pre-picked TargetPods).
func (r *Runner) startExperiment(ctx context.Context, exp domain.Experiment) {
	if _, ok := r.Actions[exp.ActionType]; !ok {
		r.Logger.Warn("unsupported action", "experiment_id", exp.ID, "action", exp.ActionType)
		r.publishFailure(ctx, exp, nil, fmt.Errorf("unsupported action %q", exp.ActionType))
		return
	}

	local, err := r.localTargets(ctx, exp)
	if err != nil {
		r.publishFailure(ctx, exp, nil, fmt.Errorf("list local pods: %w", err))
		return
	}
	if len(local) == 0 {
		// Nothing on this node — silently skip. Another agent (or none,
		// if all candidates were unschedulable) will publish the result.
		return
	}

	// Last-chance kill-switch check before we touch anything. The Kafka
	// broadcast is best-effort; the Redis read is the source of truth.
	engaged, err := r.Switch.KillSwitchEngaged(ctx)
	if err != nil {
		r.publishFailure(ctx, exp, local, fmt.Errorf("read killswitch: %w", err))
		return
	}
	if engaged {
		r.publishAborted(ctx, exp, local, "kill switch engaged")
		return
	}

	expCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.active[exp.ID] = cancel
	r.mu.Unlock()

	go func() {
		defer func() {
			r.mu.Lock()
			delete(r.active, exp.ID)
			r.mu.Unlock()
			cancel()
		}()
		r.runAction(expCtx, cancel, exp, local)
	}()
}

// runAction executes the chaos action against every locally-scheduled
// target pod and publishes one result covering all of them. A
// kill-switch poller runs alongside the action and calls expCancel if
// the switch flips during a long-running action.
func (r *Runner) runAction(expCtx context.Context, expCancel context.CancelFunc, exp domain.Experiment, local []string) {
	action := r.Actions[exp.ActionType]
	startedAt := r.Now()

	go r.pollKillSwitch(expCtx, expCancel)

	var execErr error
	for _, pod := range local {
		if err := expCtx.Err(); err != nil {
			execErr = err
			break
		}
		if err := action.Execute(expCtx, exp, pod); err != nil {
			execErr = err
			break
		}
	}

	completedAt := r.Now()
	r.publish(context.Background(), exp, local, startedAt, completedAt, execErr)
}

// pollKillSwitch polls Redis and cancels the experiment when the switch
// trips. It returns when ctx is cancelled (either because the action
// finished or because the agent itself is shutting down).
func (r *Runner) pollKillSwitch(ctx context.Context, cancel context.CancelFunc) {
	if r.PollEvery <= 0 {
		return
	}
	t := time.NewTicker(r.PollEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			engaged, err := r.Switch.KillSwitchEngaged(ctx)
			if err != nil {
				// Transient Redis errors shouldn't kill the experiment.
				r.Logger.Warn("killswitch poll failed", "err", err)
				continue
			}
			if engaged {
				r.Logger.Info("killswitch tripped during action — aborting")
				cancel()
				return
			}
		}
	}
}

// localTargets intersects the control-plane-picked TargetPods with the
// pods of this experiment's selector that are scheduled on this node.
// The intersection ensures we never act on a pod the control plane
// didn't whitelist, even if a new pod with the same labels has been
// scheduled here in the meantime.
func (r *Runner) localTargets(ctx context.Context, exp domain.Experiment) ([]string, error) {
	if len(exp.TargetPods) == 0 {
		return nil, nil
	}
	onNode, err := r.Pods.ListPodsOnNode(ctx, r.NodeName, exp.TargetNamespace, exp.TargetSelector)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]struct{}, len(exp.TargetPods))
	for _, p := range exp.TargetPods {
		allowed[p] = struct{}{}
	}
	out := make([]string, 0, len(onNode))
	for _, p := range onNode {
		if _, ok := allowed[p]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// abort cancels a single in-flight experiment's context. Safe to call
// even if the experiment was never running on this agent.
func (r *Runner) abort(id domain.ID) {
	r.mu.Lock()
	cancel, ok := r.active[id]
	r.mu.Unlock()
	if !ok {
		return
	}
	r.Logger.Info("aborting experiment", "experiment_id", id)
	cancel()
}

// abortAll cancels every in-flight experiment on this agent. Triggered
// by a kill-switch broadcast; the Redis-side check inside runAction is
// the authoritative one.
func (r *Runner) abortAll() {
	r.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(r.active))
	for _, c := range r.active {
		cancels = append(cancels, c)
	}
	r.mu.Unlock()
	r.Logger.Info("kill switch — aborting all", "count", len(cancels))
	for _, c := range cancels {
		c()
	}
}

// publish writes a single result covering every pod the agent
// attempted. Outcome is derived from execErr: nil → success, ctx
// cancelled → aborted, anything else → failure.
func (r *Runner) publish(ctx context.Context, exp domain.Experiment, pods []string, startedAt, completedAt time.Time, execErr error) {
	res := domain.ExperimentResult{
		ID:           r.NewID(),
		ExperimentID: exp.ID,
		AgentNode:    r.NodeName,
		AffectedPods: pods,
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
	}
	switch {
	case execErr == nil:
		res.Outcome = domain.OutcomeSuccess
	case errors.Is(execErr, context.Canceled), errors.Is(execErr, context.DeadlineExceeded):
		res.Outcome = domain.OutcomeAborted
		res.ErrorMessage = execErr.Error()
	default:
		res.Outcome = domain.OutcomeFailure
		res.ErrorMessage = execErr.Error()
	}

	if err := r.Results.Publish(ctx, string(exp.ID), res); err != nil {
		// We can't recover from a failed publish — the recorder will
		// never learn the experiment finished and the lock will only
		// release on TTL. Log loudly so an operator notices.
		r.Logger.Error("publish result failed",
			"experiment_id", exp.ID,
			"result_id", res.ID,
			"outcome", res.Outcome,
			"err", err,
		)
		return
	}
	r.Logger.Info("published result",
		"experiment_id", exp.ID,
		"result_id", res.ID,
		"outcome", res.Outcome,
		"affected_pods", pods,
	)
}

// publishFailure is a convenience for early-exit error paths where no
// action ran at all. Uses Now() for both start and end timestamps.
func (r *Runner) publishFailure(ctx context.Context, exp domain.Experiment, pods []string, err error) {
	t := r.Now()
	r.publish(ctx, exp, pods, t, t, err)
}

// publishAborted is the early-exit equivalent for kill-switch trips
// before the action started.
func (r *Runner) publishAborted(ctx context.Context, exp domain.Experiment, pods []string, reason string) {
	t := r.Now()
	r.publish(ctx, exp, pods, t, t, fmt.Errorf("%s: %w", reason, context.Canceled))
}
