package recorder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

type fakeResults struct {
	seen map[domain.ID]int
	err  error
}

func newFakeResults() *fakeResults { return &fakeResults{seen: map[domain.ID]int{}} }

func (f *fakeResults) InsertResult(_ context.Context, r domain.ExperimentResult) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	f.seen[r.ID]++
	return f.seen[r.ID] == 1, nil
}

type fakeExperiments struct {
	byID map[domain.ID]domain.Experiment
}

func (f fakeExperiments) GetExperiment(_ context.Context, id domain.ID) (*domain.Experiment, error) {
	if e, ok := f.byID[id]; ok {
		return &e, nil
	}
	return nil, nil
}

type fakeStatus struct {
	transitions map[domain.ID]domain.Status
}

func newFakeStatus() *fakeStatus { return &fakeStatus{transitions: map[domain.ID]domain.Status{}} }

func (f *fakeStatus) UpdateStatus(_ context.Context, id domain.ID, status domain.Status, _ string) error {
	f.transitions[id] = status
	return nil
}

type fakeLocks struct {
	calls []struct{ service, expID string }
}

func (f *fakeLocks) ReleaseLockIfOwner(_ context.Context, service, expID string) (bool, error) {
	f.calls = append(f.calls, struct{ service, expID string }{service, expID})
	return true, nil
}

type harness struct {
	rec     *Recorder
	results *fakeResults
	status  *fakeStatus
	locks   *fakeLocks
	logBuf  *bytes.Buffer
}

func newHarness(exps map[domain.ID]domain.Experiment) *harness {
	buf := &bytes.Buffer{}
	results := newFakeResults()
	status := newFakeStatus()
	locks := &fakeLocks{}
	return &harness{
		rec: &Recorder{
			Experiments: fakeExperiments{byID: exps},
			Results:     results,
			Statuses:    status,
			Locks:       locks,
			Logger:      slog.New(slog.NewJSONHandler(buf, nil)),
		},
		results: results,
		status:  status,
		locks:   locks,
		logBuf:  buf,
	}
}

func payload(t *testing.T, r domain.ExperimentResult) []byte {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func sampleExperiment() domain.Experiment {
	return domain.Experiment{
		ID:              "exp-1",
		TargetNamespace: "prod",
		TargetSelector:  map[string]string{"app": "payments"},
	}
}

func TestProcess_HappyPath(t *testing.T) {
	exp := sampleExperiment()
	h := newHarness(map[domain.ID]domain.Experiment{exp.ID: exp})

	now := time.Now().UTC()
	err := h.rec.Process(context.Background(), payload(t, domain.ExperimentResult{
		ID: "r-1", ExperimentID: exp.ID,
		AgentNode: "node-a", Outcome: domain.OutcomeSuccess,
		StartedAt: now.Add(-5 * time.Second), CompletedAt: now,
	}))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if h.results.seen["r-1"] != 1 {
		t.Errorf("expected result inserted once")
	}
	if got := h.status.transitions[exp.ID]; got != domain.StatusCompleted {
		t.Errorf("status: got %q, want completed", got)
	}
	if len(h.locks.calls) != 1 || h.locks.calls[0].service != "prod/payments" || h.locks.calls[0].expID != "exp-1" {
		t.Errorf("expected one ReleaseLockIfOwner(prod/payments, exp-1); got %+v", h.locks.calls)
	}
}

func TestProcess_IdempotentOnDuplicate(t *testing.T) {
	exp := sampleExperiment()
	h := newHarness(map[domain.ID]domain.Experiment{exp.ID: exp})
	body := payload(t, domain.ExperimentResult{
		ID: "r-1", ExperimentID: exp.ID, Outcome: domain.OutcomeSuccess,
	})
	for i := 0; i < 3; i++ {
		if err := h.rec.Process(context.Background(), body); err != nil {
			t.Fatalf("delivery %d: %v", i, err)
		}
	}
	if h.results.seen["r-1"] != 3 {
		t.Errorf("InsertResult called %d times, expected 3", h.results.seen["r-1"])
	}
	// Status update is set-to-same-value so it's safe to repeat.
	if got := h.status.transitions[exp.ID]; got != domain.StatusCompleted {
		t.Errorf("final status: got %q", got)
	}
}

func TestProcess_AbortedOutcomeMapsToAbortedStatus(t *testing.T) {
	exp := sampleExperiment()
	h := newHarness(map[domain.ID]domain.Experiment{exp.ID: exp})
	err := h.rec.Process(context.Background(), payload(t, domain.ExperimentResult{
		ID: "r-1", ExperimentID: exp.ID, Outcome: domain.OutcomeAborted,
	}))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if got := h.status.transitions[exp.ID]; got != domain.StatusAborted {
		t.Errorf("status: got %q, want aborted", got)
	}
}

func TestProcess_FailureOutcomeMapsToCompleted(t *testing.T) {
	exp := sampleExperiment()
	h := newHarness(map[domain.ID]domain.Experiment{exp.ID: exp})
	err := h.rec.Process(context.Background(), payload(t, domain.ExperimentResult{
		ID: "r-1", ExperimentID: exp.ID, Outcome: domain.OutcomeFailure,
		ErrorMessage: "tc setup failed",
	}))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if got := h.status.transitions[exp.ID]; got != domain.StatusCompleted {
		t.Errorf("status: got %q, want completed", got)
	}
}

func TestProcess_UnknownExperimentRecordsResultButSkipsStatusAndLock(t *testing.T) {
	h := newHarness(nil)
	err := h.rec.Process(context.Background(), payload(t, domain.ExperimentResult{
		ID: "r-1", ExperimentID: "ghost", Outcome: domain.OutcomeSuccess,
	}))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if h.results.seen["r-1"] != 1 {
		t.Errorf("result should still be recorded for forensics")
	}
	if len(h.status.transitions) != 0 {
		t.Errorf("no status transition expected; got %v", h.status.transitions)
	}
	if len(h.locks.calls) != 0 {
		t.Errorf("no lock release expected; got %v", h.locks.calls)
	}
}

func TestProcess_BadJSON(t *testing.T) {
	h := newHarness(nil)
	if err := h.rec.Process(context.Background(), []byte("not json")); err == nil {
		t.Fatal("expected error on malformed payload")
	}
}

func TestProcess_RejectsMissingIDs(t *testing.T) {
	h := newHarness(nil)
	body := payload(t, domain.ExperimentResult{Outcome: domain.OutcomeSuccess})
	if err := h.rec.Process(context.Background(), body); err == nil {
		t.Fatal("expected error when ids are missing")
	}
}

func TestProcess_PropagatesInsertError(t *testing.T) {
	boom := errors.New("db down")
	h := newHarness(nil)
	h.results.err = boom
	body := payload(t, domain.ExperimentResult{
		ID: "r-1", ExperimentID: "exp-1", Outcome: domain.OutcomeSuccess,
	})
	if err := h.rec.Process(context.Background(), body); !errors.Is(err, boom) {
		t.Fatalf("got %v, want db down", err)
	}
}
