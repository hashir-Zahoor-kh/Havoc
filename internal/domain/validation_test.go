package domain

import (
	"errors"
	"testing"
	"time"
)

func validExperiment() Experiment {
	return Experiment{
		ActionType:      ActionPodKill,
		TargetSelector:  map[string]string{"app": "payments"},
		TargetNamespace: "prod",
		DurationSeconds: 60,
	}
}

func TestValidate_OK(t *testing.T) {
	if err := validExperiment().Validate(time.Now()); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidate_UnknownAction(t *testing.T) {
	e := validExperiment()
	e.ActionType = "nuke_from_orbit"
	if err := e.Validate(time.Now()); !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("got %v, want ErrInvalidAction", err)
	}
}

func TestValidate_MissingSelector(t *testing.T) {
	e := validExperiment()
	e.TargetSelector = nil
	if err := e.Validate(time.Now()); !errors.Is(err, ErrMissingSelector) {
		t.Fatalf("got %v, want ErrMissingSelector", err)
	}
}

func TestValidate_MissingNamespace(t *testing.T) {
	e := validExperiment()
	e.TargetNamespace = ""
	if err := e.Validate(time.Now()); !errors.Is(err, ErrMissingNamespace) {
		t.Fatalf("got %v, want ErrMissingNamespace", err)
	}
}

func TestValidate_DurationBounds(t *testing.T) {
	for _, d := range []int{0, -1, MaxDurationSeconds + 1} {
		e := validExperiment()
		e.DurationSeconds = d
		if err := e.Validate(time.Now()); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("duration %d: got %v, want ErrInvalidDuration", d, err)
		}
	}
}

func TestValidate_ScheduleTooFar(t *testing.T) {
	now := time.Now()
	e := validExperiment()
	e.ScheduledFor = now.Add(MaxScheduleHorizon + time.Hour)
	if err := e.Validate(now); !errors.Is(err, ErrInvalidSchedule) {
		t.Fatalf("got %v, want ErrInvalidSchedule", err)
	}
}

func TestValidate_ScheduleWithinHorizon(t *testing.T) {
	now := time.Now()
	e := validExperiment()
	e.ScheduledFor = now.Add(time.Hour)
	if err := e.Validate(now); err != nil {
		t.Fatalf("within horizon should pass, got %v", err)
	}
}

func TestValidate_LatencyRequired(t *testing.T) {
	e := validExperiment()
	e.ActionType = ActionNetworkLatency
	if err := e.Validate(time.Now()); !errors.Is(err, ErrInvalidLatency) {
		t.Fatalf("missing latency: got %v, want ErrInvalidLatency", err)
	}
	e.Parameters.LatencyMilliseconds = 500
	if err := e.Validate(time.Now()); err != nil {
		t.Fatalf("valid latency should pass, got %v", err)
	}
}

func TestValidate_CPUPressureDefault(t *testing.T) {
	e := validExperiment()
	e.ActionType = ActionCPUPressure
	if err := e.Validate(time.Now()); err != nil {
		t.Fatalf("default cpu percent should be accepted, got %v", err)
	}
	e.Parameters.CPUPercent = 150
	if err := e.Validate(time.Now()); !errors.Is(err, ErrInvalidCPUPercent) {
		t.Fatalf("out-of-range cpu percent: got %v, want ErrInvalidCPUPercent", err)
	}
}

func TestServiceName_WithAppLabel(t *testing.T) {
	if got := validExperiment().ServiceName(); got != "prod/payments" {
		t.Fatalf("got %q, want prod/payments", got)
	}
}

func TestServiceName_FallsBackToNamespace(t *testing.T) {
	e := validExperiment()
	e.TargetSelector = map[string]string{"tier": "backend"}
	if got := e.ServiceName(); got != "prod" {
		t.Fatalf("got %q, want prod", got)
	}
}

func TestActionType_Valid(t *testing.T) {
	for _, a := range []ActionType{ActionPodKill, ActionNetworkLatency, ActionCPUPressure} {
		if !a.Valid() {
			t.Errorf("%q should be valid", a)
		}
	}
	if ActionType("").Valid() || ActionType("bogus").Valid() {
		t.Errorf("empty/unknown action types should be invalid")
	}
}
