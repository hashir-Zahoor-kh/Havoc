package safety

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

type fakeKill struct {
	engaged bool
	err     error
}

func (f fakeKill) KillSwitchEngaged(context.Context) (bool, error) { return f.engaged, f.err }

type fakeLocks struct {
	locked map[string]bool
	err    error
}

func (f fakeLocks) IsLocked(_ context.Context, service string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.locked[service], nil
}

type fakePods struct {
	pods map[string][]string
	err  error
}

func (f fakePods) ListPods(_ context.Context, namespace string, _ map[string]string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pods[namespace], nil
}

type fakeBlackouts struct {
	windows []BlackoutWindow
	err     error
}

func (f fakeBlackouts) ListWindows(context.Context) ([]BlackoutWindow, error) {
	return f.windows, f.err
}

func baseGuard(opts ...func(*Guard)) *Guard {
	g := &Guard{
		KillSwitch: fakeKill{engaged: false},
		Locks:      fakeLocks{locked: map[string]bool{}},
		Pods:       fakePods{pods: map[string][]string{"prod": {"a", "b", "c", "d"}}},
		Blackouts:  fakeBlackouts{},
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

func baseExperiment() domain.Experiment {
	return domain.Experiment{
		ActionType:      domain.ActionPodKill,
		TargetSelector:  map[string]string{"app": "payments"},
		TargetNamespace: "prod",
		DurationSeconds: 60,
	}
}

func TestGuard_AllowsValid(t *testing.T) {
	if err := baseGuard().Check(context.Background(), baseExperiment(), time.Now()); err != nil {
		t.Fatalf("expected allow, got %v", err)
	}
}

func TestGuard_RejectsKillSwitch(t *testing.T) {
	g := baseGuard(func(g *Guard) { g.KillSwitch = fakeKill{engaged: true} })
	err := g.Check(context.Background(), baseExperiment(), time.Now())
	if !errors.Is(err, ErrKillSwitchEngaged) {
		t.Fatalf("got %v, want ErrKillSwitchEngaged", err)
	}
}

func TestGuard_RejectsActiveLock(t *testing.T) {
	g := baseGuard(func(g *Guard) {
		g.Locks = fakeLocks{locked: map[string]bool{"prod/payments": true}}
	})
	err := g.Check(context.Background(), baseExperiment(), time.Now())
	if !errors.Is(err, ErrActiveExperiment) {
		t.Fatalf("got %v, want ErrActiveExperiment", err)
	}
}

func TestGuard_RejectsNoMatchingPods(t *testing.T) {
	g := baseGuard(func(g *Guard) {
		g.Pods = fakePods{pods: map[string][]string{}}
	})
	err := g.Check(context.Background(), baseExperiment(), time.Now())
	if !errors.Is(err, ErrNoMatchingPods) {
		t.Fatalf("got %v, want ErrNoMatchingPods", err)
	}
}

func TestGuard_RejectsBlastRadius(t *testing.T) {
	// 3 pods at default 25%: 1/3 ≈ 33% > 25% → reject.
	g := baseGuard(func(g *Guard) {
		g.Pods = fakePods{pods: map[string][]string{"prod": {"a", "b", "c"}}}
	})
	err := g.Check(context.Background(), baseExperiment(), time.Now())
	if !errors.Is(err, ErrBlastRadiusExceeded) {
		t.Fatalf("got %v, want ErrBlastRadiusExceeded", err)
	}
}

func TestGuard_AllowsAtBlastRadiusBoundary(t *testing.T) {
	// 4 pods at 25%: 1/4 = 25% → allow.
	g := baseGuard()
	if err := g.Check(context.Background(), baseExperiment(), time.Now()); err != nil {
		t.Fatalf("4-pod case should pass at 25%%, got %v", err)
	}
}

func TestGuard_CustomBlastRadius(t *testing.T) {
	// 2 pods at 50%: 1/2 = 50% → allow.
	g := baseGuard(func(g *Guard) {
		g.Config.MaxBlastRadiusPercent = 50
		g.Pods = fakePods{pods: map[string][]string{"prod": {"a", "b"}}}
	})
	if err := g.Check(context.Background(), baseExperiment(), time.Now()); err != nil {
		t.Fatalf("2-pod case at 50%% should pass, got %v", err)
	}
}

func TestGuard_RejectsInsideBlackout(t *testing.T) {
	// Window fires every minute and lasts 2 minutes → always active.
	g := baseGuard(func(g *Guard) {
		g.Blackouts = fakeBlackouts{windows: []BlackoutWindow{{
			Name:           "always-on",
			CronExpression: "* * * * *",
			Duration:       2 * time.Minute,
		}}}
	})
	err := g.Check(context.Background(), baseExperiment(), time.Now())
	if !errors.Is(err, ErrInBlackoutWindow) {
		t.Fatalf("got %v, want ErrInBlackoutWindow", err)
	}
}

func TestBlackoutWindow_ActiveDuringBusinessHours(t *testing.T) {
	// Business hours: 9am weekdays for 8 hours.
	w := BlackoutWindow{
		Name:           "business-hours",
		CronExpression: "0 9 * * 1-5",
		Duration:       8 * time.Hour,
	}
	// Monday 2026-01-05 at 10:00 — inside the window.
	inside := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	active, err := w.Active(inside)
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if !active {
		t.Fatalf("expected active during business hours")
	}
	// Monday 2026-01-05 at 20:00 — past the 8h window.
	outside := time.Date(2026, 1, 5, 20, 0, 0, 0, time.UTC)
	active, err = w.Active(outside)
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if active {
		t.Fatalf("expected inactive outside business hours")
	}
	// Saturday 2026-01-10 at 10:00 — weekend, never fires.
	weekend := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	active, err = w.Active(weekend)
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if active {
		t.Fatalf("expected inactive on weekend")
	}
}

func TestGuard_PropagatesDependencyError(t *testing.T) {
	boom := errors.New("boom")
	g := baseGuard(func(g *Guard) { g.KillSwitch = fakeKill{err: boom} })
	if err := g.Check(context.Background(), baseExperiment(), time.Now()); !errors.Is(err, boom) {
		t.Fatalf("expected wrapped boom, got %v", err)
	}
}
