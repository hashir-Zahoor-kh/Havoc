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

// NetworkLatency injects N milliseconds of outbound latency on the
// target pod's eth0 using `tc qdisc add ... netem delay`. The qdisc is
// removed when the configured duration elapses or the experiment is
// aborted.
//
// This action requires the target pod to ship `iproute2` (so `tc` is
// available inside the container) and to run with NET_ADMIN — which is
// not the default. For a controlled demo we pin a sample workload that
// satisfies both. The brief acknowledges this is the hardest of the
// three actions; if a deployment can't grant NET_ADMIN, fall back to
// the pod_kill or cpu_pressure actions.
type NetworkLatency struct {
	K8s ExecRunner
}

func (n *NetworkLatency) Type() domain.ActionType { return domain.ActionNetworkLatency }

func (n *NetworkLatency) Execute(ctx context.Context, exp domain.Experiment, podName string) error {
	latencyMS := exp.Parameters.LatencyMilliseconds
	if latencyMS <= 0 {
		return fmt.Errorf("network_latency requires latency_ms > 0")
	}
	container, err := n.K8s.FirstContainerName(ctx, exp.TargetNamespace, podName)
	if err != nil {
		return fmt.Errorf("resolve container: %w", err)
	}

	// Add the qdisc, sleep for the duration, then remove it. The trap
	// covers the abort case so we don't strand a netem qdisc on the
	// target's NIC if the agent's exec call is killed mid-stream.
	script := fmt.Sprintf(`
set -eu
cleanup() { tc qdisc del dev eth0 root 2>/dev/null || true; }
trap cleanup EXIT INT TERM
tc qdisc add dev eth0 root netem delay %dms
sleep %d
cleanup
`, latencyMS, exp.DurationSeconds)

	var stderr bytes.Buffer
	err = n.K8s.ExecInPod(ctx, k8s.ExecOptions{
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
		return fmt.Errorf("network latency exec: %w (stderr: %s)", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return nil
}
