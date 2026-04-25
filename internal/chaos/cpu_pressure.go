package chaos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
	"github.com/hashir-Zahoor-kh/Havoc/internal/k8s"
)

// CPUPressure spins busy-loops inside the target container for the
// experiment's duration. The burn happens against the target pod's own
// cgroup, so it counts toward its CPU quota and exercises real
// throttling / autoscaling paths.
//
// Image requirements: a POSIX shell (`sh`) and either `nproc` or
// `getconf _NPROCESSORS_ONLN`. Most distroless images do not satisfy
// this — for the demo workloads we ship, alpine or busybox is fine.
type CPUPressure struct {
	K8s ExecRunner
}

// ExecRunner is the slice of the Kubernetes client used by exec-based
// actions. Defining it here keeps the chaos package decoupled from
// client-go for testing.
type ExecRunner interface {
	ExecInPod(ctx context.Context, opts k8s.ExecOptions) error
	FirstContainerName(ctx context.Context, namespace, pod string) (string, error)
}

func (c *CPUPressure) Type() domain.ActionType { return domain.ActionCPUPressure }

func (c *CPUPressure) Execute(ctx context.Context, exp domain.Experiment, podName string) error {
	pct := exp.Parameters.CPUPercent
	if pct <= 0 {
		pct = domain.DefaultCPUPercent
	}
	container, err := c.K8s.FirstContainerName(ctx, exp.TargetNamespace, podName)
	if err != nil {
		return fmt.Errorf("resolve container: %w", err)
	}

	// One busy-looping subshell per CPU. The duty-cycle approximation
	// uses sleep to throttle: <pct>ms of work, <100-pct>ms of sleep,
	// repeated for the configured duration.
	script := fmt.Sprintf(`
set -eu
duration=%d
pct=%d
N=$(nproc 2>/dev/null || getconf _NPROCESSORS_ONLN 2>/dev/null || echo 1)
end=$(( $(date +%%s) + duration ))
for i in $(seq 1 "$N"); do
  (
    while [ "$(date +%%s)" -lt "$end" ]; do
      tick_end=$(( $(date +%%s%%N) / 1000000 + pct ))
      while [ "$(( $(date +%%s%%N) / 1000000 ))" -lt "$tick_end" ]; do :; done
      [ "$pct" -lt 100 ] && sleep "$(awk "BEGIN { print (100 - $pct) / 1000 }")"
    done
  ) &
done
wait
`, exp.DurationSeconds, pct)

	var stderr bytes.Buffer
	err = c.K8s.ExecInPod(ctx, k8s.ExecOptions{
		Namespace: exp.TargetNamespace,
		Pod:       podName,
		Container: container,
		Command:   []string{"sh", "-c", script},
		Stdout:    io.Discard,
		Stderr:    &stderr,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("cpu pressure exec: %w (stderr: %s)", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return nil
}
