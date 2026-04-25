// Package k8s wraps client-go to provide the narrow set of operations
// Havoc actually needs: listing pods by selector (for blast radius checks
// in the control plane), deleting pods (for the pod_kill action), and
// executing commands inside a target container (for cpu_pressure and
// network_latency). Keeping this narrow prevents the k8s import surface
// from leaking into unrelated packages.
package k8s

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// Client is a typed wrapper around a Kubernetes clientset and the rest
// config it was built from. The rest config is retained so the exec
// subresource can construct its own SPDY transport.
type Client struct {
	cs      *kubernetes.Clientset
	restCfg *rest.Config
}

// Config selects how the Kubernetes client is constructed. InCluster uses
// the pod's service-account credentials; otherwise KubeconfigPath is used
// (empty path falls back to the default KUBECONFIG resolution).
type Config struct {
	InCluster      bool
	KubeconfigPath string
}

// New constructs a client based on the given config.
func New(cfg Config) (*Client, error) {
	restCfg, err := buildRESTConfig(cfg)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("build clientset: %w", err)
	}
	return &Client{cs: cs, restCfg: restCfg}, nil
}

func buildRESTConfig(cfg Config) (*rest.Config, error) {
	if cfg.InCluster {
		c, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("in-cluster config: %w", err)
		}
		return c, nil
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if cfg.KubeconfigPath != "" {
		rules.ExplicitPath = cfg.KubeconfigPath
	}
	kc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	c, err := kc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("kubeconfig: %w", err)
	}
	return c, nil
}

// ListPods returns the names of pods in namespace that match selector.
// Implements safety.PodLister.
func (c *Client) ListPods(ctx context.Context, namespace string, selector map[string]string) ([]string, error) {
	list, err := c.cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector).String(),
	})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	names := make([]string, 0, len(list.Items))
	for _, p := range list.Items {
		names = append(names, p.Name)
	}
	return names, nil
}

// DeletePod deletes a single pod. The agent calls this for the pod_kill
// action.
func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	if err := c.cs.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("delete pod %s/%s: %w", namespace, name, err)
	}
	return nil
}

// ListPodsOnNode returns pods in namespace matching selector that are
// assigned to the named node. The agent uses this to filter commands to
// those whose targets live on its own node.
func (c *Client) ListPodsOnNode(ctx context.Context, node, namespace string, selector map[string]string) ([]string, error) {
	list, err := c.cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector).String(),
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", node),
	})
	if err != nil {
		return nil, fmt.Errorf("list pods on node %s: %w", node, err)
	}
	names := make([]string, 0, len(list.Items))
	for _, p := range list.Items {
		names = append(names, p.Name)
	}
	return names, nil
}

// FirstContainerName returns the name of the pod's first container. The
// agent uses this when the experiment doesn't pin a container — most
// workloads have just one.
func (c *Client) FirstContainerName(ctx context.Context, namespace, pod string) (string, error) {
	p, err := c.cs.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get pod %s/%s: %w", namespace, pod, err)
	}
	if len(p.Spec.Containers) == 0 {
		return "", fmt.Errorf("pod %s/%s has no containers", namespace, pod)
	}
	return p.Spec.Containers[0].Name, nil
}

// ExecOptions targets a single command execution inside a container.
type ExecOptions struct {
	Namespace string
	Pod       string
	Container string
	Command   []string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
}

// ExecInPod runs Command inside the named container and streams stdout
// and stderr through the supplied writers. It returns when the command
// exits, when the context is cancelled (the agent uses this for abort),
// or when the connection breaks.
func (c *Client) ExecInPod(ctx context.Context, opts ExecOptions) error {
	req := c.cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(opts.Pod).
		Namespace(opts.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: opts.Container,
			Command:   opts.Command,
			Stdin:     opts.Stdin != nil,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.restCfg, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("build exec: %w", err)
	}
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  opts.Stdin,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	}); err != nil {
		return fmt.Errorf("stream exec: %w", err)
	}
	return nil
}
