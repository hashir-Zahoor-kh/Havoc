// Package k8s wraps client-go to provide the narrow set of operations
// Havoc actually needs: listing pods by selector (for blast radius checks
// in the control plane) and deleting pods (for the pod_kill action in the
// agent). Keeping this narrow prevents the k8s import surface from
// leaking into unrelated packages.
package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is a typed wrapper around a Kubernetes clientset.
type Client struct {
	cs *kubernetes.Clientset
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
	return &Client{cs: cs}, nil
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
