package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashir-Zahoor-kh/Havoc/internal/chaos"
	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

// fakePods feeds the runner a deterministic node-local pod list.
type fakePods struct {
	pods map[string][]string // namespace -> pods on node
	err  error
}

func (f *fakePods) ListPodsOnNode(_ context.Context, _ string, namespace string, _ map[string]string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pods[namespace], nil
}

// fakeSwitch is a kill-switch the test can flip.
type fakeSwitch struct {
	engaged atomic.Bool
	err     error
}

func (f *fakeSwitch) KillSwitchEngaged(_ context.Context) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.engaged.Load(), nil
}

// fakePublisher captures every result the runner publishes.
type fakePublisher struct {
	mu      sync.Mutex
	results []domain.ExperimentResult
	err     error
}

func (f *fakePublisher) Publish(_ context.Context, _ string, v any) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := v.(domain.ExperimentResult); ok {
		f.results = append(f.results, r)
	}
	return nil
}

func (f *fakePublisher) all() []domain.ExperimentResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.ExperimentResult, len(f.results))
	copy(out, f.results)
	return out
}

// fakeAction records calls and optionally blocks/errors.
type fakeAction struct {
	typ      domain.ActionType
	calls    atomic.Int32
	pods     []string
	mu       sync.Mutex
	err      error
	block    chan struct{} // if non-nil, Execute blocks on it / ctx.Done
	respects bool          // true: respects ctx cancellation while blocked
}

func (f *fakeAction) Type() domain.ActionType { return f.typ }

func (f *fakeAction) Execute(ctx context.Context, _ domain.Experiment, podName string) error {
	f.calls.Add(1)
	f.mu.Lock()
	f.pods = append(f.pods, podName)
	f.mu.Unlock()
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			if f.respects {
				return ctx.Err()
			}
		}
	}
	return f.err
}

func (f *fakeAction) seenPods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.pods))
	copy(out, f.pods)
	return out
}

func newRunner(t *testing.T, pods *fakePods, sw *fakeSwitch, pub *fakePublisher, action *fakeAction) *Runner {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := New("node-a", pods, sw, pub, chaos.NewRegistry(action), logger)
	// Fix time and id so result comparisons are deterministic.
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	r.Now = func() time.Time { return now }
	var n atomic.Int64
	r.NewID = func() domain.ID {
		return domain.ID(fmt.Sprintf("res-%d", n.Add(1)))
	}
	// Keep poll cadence tight so kill-switch tests don't drag.
	r.PollEvery = 5 * time.Millisecond
	return r
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for: %s", msg)
}

func baseExp() domain.Experiment {
	return domain.Experiment{
		ID:              "exp-1",
		ActionType:      domain.ActionPodKill,
		TargetNamespace: "demo",
		TargetSelector:  map[string]string{"app": "checkout"},
		TargetPods:      []string{"pod-a", "pod-b"},
		DurationSeconds: 5,
	}
}

func encodeSchedule(t *testing.T, exp domain.Experiment) []byte {
	t.Helper()
	cmd := domain.Command{Type: domain.CommandSchedule, Experiment: &exp}
	return mustJSON(t, cmd)
}

func encodeAbort(t *testing.T, id domain.ID) []byte {
	t.Helper()
	return mustJSON(t, domain.Command{Type: domain.CommandAbort, ExperimentID: id})
}

func encodeKill(t *testing.T) []byte {
	t.Helper()
	return mustJSON(t, domain.Command{Type: domain.CommandKillSwitch})
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestSchedule_HappyPath(t *testing.T) {
	pods := &fakePods{pods: map[string][]string{"demo": {"pod-a", "pod-c"}}}
	sw := &fakeSwitch{}
	pub := &fakePublisher{}
	act := &fakeAction{typ: domain.ActionPodKill}
	r := newRunner(t, pods, sw, pub, act)

	if err := r.Handle(context.Background(), encodeSchedule(t, baseExp())); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	waitFor(t, func() bool { return len(pub.all()) == 1 }, "result published")

	got := pub.all()[0]
	if got.Outcome != domain.OutcomeSuccess {
		t.Fatalf("outcome = %s, want success", got.Outcome)
	}
	// pod-a is the only intersection of TargetPods and node-local pods.
	if len(got.AffectedPods) != 1 || got.AffectedPods[0] != "pod-a" {
		t.Fatalf("affected pods = %v, want [pod-a]", got.AffectedPods)
	}
	if act.calls.Load() != 1 {
		t.Fatalf("action calls = %d, want 1", act.calls.Load())
	}
	if got.AgentNode != "node-a" {
		t.Fatalf("agent node = %s, want node-a", got.AgentNode)
	}
}

func TestSchedule_NoLocalPods_NoResult(t *testing.T) {
	pods := &fakePods{pods: map[string][]string{"demo": {"pod-z"}}} // not in TargetPods
	sw := &fakeSwitch{}
	pub := &fakePublisher{}
	act := &fakeAction{typ: domain.ActionPodKill}
	r := newRunner(t, pods, sw, pub, act)

	if err := r.Handle(context.Background(), encodeSchedule(t, baseExp())); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	// Give a brief window to detect any spurious publish.
	time.Sleep(20 * time.Millisecond)
	if got := len(pub.all()); got != 0 {
		t.Fatalf("expected no result, got %d", got)
	}
	if act.calls.Load() != 0 {
		t.Fatalf("expected no action calls, got %d", act.calls.Load())
	}
}

func TestSchedule_KillSwitchPreCheck_Aborts(t *testing.T) {
	pods := &fakePods{pods: map[string][]string{"demo": {"pod-a"}}}
	sw := &fakeSwitch{}
	sw.engaged.Store(true)
	pub := &fakePublisher{}
	act := &fakeAction{typ: domain.ActionPodKill}
	r := newRunner(t, pods, sw, pub, act)

	if err := r.Handle(context.Background(), encodeSchedule(t, baseExp())); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	waitFor(t, func() bool { return len(pub.all()) == 1 }, "aborted result")
	if got := pub.all()[0].Outcome; got != domain.OutcomeAborted {
		t.Fatalf("outcome = %s, want aborted", got)
	}
	if act.calls.Load() != 0 {
		t.Fatalf("action ran despite kill switch")
	}
}

func TestSchedule_ActionFailure_PublishesFailure(t *testing.T) {
	pods := &fakePods{pods: map[string][]string{"demo": {"pod-a"}}}
	sw := &fakeSwitch{}
	pub := &fakePublisher{}
	act := &fakeAction{typ: domain.ActionPodKill, err: errors.New("boom")}
	r := newRunner(t, pods, sw, pub, act)

	if err := r.Handle(context.Background(), encodeSchedule(t, baseExp())); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	waitFor(t, func() bool { return len(pub.all()) == 1 }, "failure result")
	got := pub.all()[0]
	if got.Outcome != domain.OutcomeFailure {
		t.Fatalf("outcome = %s, want failure", got.Outcome)
	}
	if got.ErrorMessage != "boom" {
		t.Fatalf("error message = %q, want boom", got.ErrorMessage)
	}
}

func TestSchedule_UnsupportedAction_PublishesFailure(t *testing.T) {
	pods := &fakePods{pods: map[string][]string{"demo": {"pod-a"}}}
	sw := &fakeSwitch{}
	pub := &fakePublisher{}
	// Registry only has pod_kill — schedule a network_latency.
	act := &fakeAction{typ: domain.ActionPodKill}
	r := newRunner(t, pods, sw, pub, act)

	exp := baseExp()
	exp.ActionType = domain.ActionNetworkLatency
	if err := r.Handle(context.Background(), encodeSchedule(t, exp)); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	waitFor(t, func() bool { return len(pub.all()) == 1 }, "failure result")
	if got := pub.all()[0].Outcome; got != domain.OutcomeFailure {
		t.Fatalf("outcome = %s, want failure", got)
	}
}

func TestKillSwitch_AbortsInFlight(t *testing.T) {
	pods := &fakePods{pods: map[string][]string{"demo": {"pod-a"}}}
	sw := &fakeSwitch{}
	pub := &fakePublisher{}
	act := &fakeAction{
		typ:      domain.ActionCPUPressure,
		block:    make(chan struct{}),
		respects: true,
	}
	r := newRunner(t, pods, sw, pub, act)

	exp := baseExp()
	exp.ActionType = domain.ActionCPUPressure
	if err := r.Handle(context.Background(), encodeSchedule(t, exp)); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	// Wait for the action to actually start before flipping the switch,
	// so we exercise the polling path rather than the pre-check.
	waitFor(t, func() bool { return act.calls.Load() == 1 }, "action started")
	sw.engaged.Store(true)
	waitFor(t, func() bool { return len(pub.all()) == 1 }, "aborted result")
	if got := pub.all()[0].Outcome; got != domain.OutcomeAborted {
		t.Fatalf("outcome = %s, want aborted", got)
	}
	close(act.block)
}

func TestAbort_CancelsSpecificExperiment(t *testing.T) {
	pods := &fakePods{pods: map[string][]string{"demo": {"pod-a"}}}
	sw := &fakeSwitch{}
	pub := &fakePublisher{}
	act := &fakeAction{
		typ:      domain.ActionCPUPressure,
		block:    make(chan struct{}),
		respects: true,
	}
	r := newRunner(t, pods, sw, pub, act)

	exp := baseExp()
	exp.ActionType = domain.ActionCPUPressure
	if err := r.Handle(context.Background(), encodeSchedule(t, exp)); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	waitFor(t, func() bool { return act.calls.Load() == 1 }, "action started")
	if err := r.Handle(context.Background(), encodeAbort(t, exp.ID)); err != nil {
		t.Fatalf("abort: %v", err)
	}
	waitFor(t, func() bool { return len(pub.all()) == 1 }, "aborted result")
	if got := pub.all()[0].Outcome; got != domain.OutcomeAborted {
		t.Fatalf("outcome = %s, want aborted", got)
	}
	close(act.block)
}

func TestKillSwitchCommand_AbortsAll(t *testing.T) {
	pods := &fakePods{pods: map[string][]string{"demo": {"pod-a"}}}
	sw := &fakeSwitch{}
	pub := &fakePublisher{}
	act := &fakeAction{
		typ:      domain.ActionCPUPressure,
		block:    make(chan struct{}),
		respects: true,
	}
	r := newRunner(t, pods, sw, pub, act)

	exp := baseExp()
	exp.ActionType = domain.ActionCPUPressure
	if err := r.Handle(context.Background(), encodeSchedule(t, exp)); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	waitFor(t, func() bool { return act.calls.Load() == 1 }, "action started")
	if err := r.Handle(context.Background(), encodeKill(t)); err != nil {
		t.Fatalf("kill: %v", err)
	}
	waitFor(t, func() bool { return len(pub.all()) == 1 }, "aborted result")
	if got := pub.all()[0].Outcome; got != domain.OutcomeAborted {
		t.Fatalf("outcome = %s, want aborted", got)
	}
	close(act.block)
}

func TestHandle_PoisonMessage_NoError(t *testing.T) {
	pods := &fakePods{}
	sw := &fakeSwitch{}
	pub := &fakePublisher{}
	act := &fakeAction{typ: domain.ActionPodKill}
	r := newRunner(t, pods, sw, pub, act)

	if err := r.Handle(context.Background(), []byte("not json")); err != nil {
		t.Fatalf("expected nil for poison message, got %v", err)
	}
	if len(pub.all()) != 0 || act.calls.Load() != 0 {
		t.Fatalf("poison message produced side effects")
	}
}
