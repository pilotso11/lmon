package k8snodes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sProvider fetches node statuses from the Kubernetes API using client-go.
type K8sProvider struct {
	client kubernetes.Interface
}

// NewK8sProvider creates a new K8sProvider with the given clientset.
func NewK8sProvider(client kubernetes.Interface) *K8sProvider {
	return &K8sProvider{client: client}
}

// GetNodeStatuses returns the status of all nodes in the cluster.
func (p *K8sProvider) GetNodeStatuses(ctx context.Context) ([]NodeStatus, error) {
	nodeList, err := p.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	statuses := make([]NodeStatus, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		ns := NodeStatus{
			Name:     node.Name,
			Cordoned: node.Spec.Unschedulable,
		}
		for _, cond := range node.Status.Conditions {
			isTrue := cond.Status == corev1.ConditionTrue
			switch cond.Type {
			case corev1.NodeReady:
				ns.Ready = isTrue
			case corev1.NodeMemoryPressure:
				ns.MemoryPressure = isTrue
			case corev1.NodeDiskPressure:
				ns.DiskPressure = isTrue
			case corev1.NodePIDPressure:
				ns.PIDPressure = isTrue
			}
		}
		statuses = append(statuses, ns)
	}

	return statuses, nil
}
