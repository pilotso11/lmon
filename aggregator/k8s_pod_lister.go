package aggregator

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sPodLister discovers lmon node pods via the official Kubernetes client-go library.
type K8sPodLister struct {
	client    kubernetes.Interface
	namespace string // empty = all namespaces
}

// NewK8sPodLister creates a PodLister using an existing Kubernetes clientset.
// namespace restricts discovery to a single namespace; empty means all namespaces.
func NewK8sPodLister(client kubernetes.Interface, namespace string) *K8sPodLister {
	return &K8sPodLister{
		client:    client,
		namespace: namespace,
	}
}

// ListPods queries the Kubernetes API for pods matching the given label selector.
// Only Running pods with a PodIP are returned.
func (l *K8sPodLister) ListPods(ctx context.Context, label string) ([]NodeEndpoint, error) {
	podList, err := l.client.CoreV1().Pods(l.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	endpoints := make([]NodeEndpoint, 0, len(podList.Items))
	for _, pod := range podList.Items {
		if pod.Status.PodIP == "" || pod.Status.Phase != "Running" {
			continue
		}
		endpoints = append(endpoints, NodeEndpoint{
			Name:    pod.Name,
			Address: pod.Status.PodIP,
		})
	}

	return endpoints, nil
}
