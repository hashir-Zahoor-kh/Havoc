package chaos

import (
	"context"

	"github.com/hashir-Zahoor-kh/Havoc/internal/domain"
)

// PodKiller deletes the target pod via the Kubernetes API. Kubernetes'
// own restart machinery brings a fresh pod back; the action under test
// is the surrounding system's behaviour during the gap.
type PodKiller struct {
	K8s PodDeleter
}

// PodDeleter is the narrow slice of the Kubernetes client this action
// needs. Defining it here lets tests inject a fake without pulling in
// client-go.
type PodDeleter interface {
	DeletePod(ctx context.Context, namespace, name string) error
}

func (p *PodKiller) Type() domain.ActionType { return domain.ActionPodKill }

func (p *PodKiller) Execute(ctx context.Context, exp domain.Experiment, podName string) error {
	return p.K8s.DeletePod(ctx, exp.TargetNamespace, podName)
}
