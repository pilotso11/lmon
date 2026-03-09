package k8sevents

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// failureReasons are the event reasons that indicate pod failures.
var failureReasons = map[string]bool{
	"CrashLoopBackOff": true,
	"OOMKilled":        true,
	"Failed":           true,
	"BackOff":          true,
	"Unhealthy":        true,
	"FailedMount":      true,
	"FailedScheduling": true,
	"Evicted":          true,
}

// K8sProvider fetches failure events from the Kubernetes API using client-go.
type K8sProvider struct {
	client kubernetes.Interface
}

// NewK8sProvider creates a new K8sProvider with the given clientset.
func NewK8sProvider(client kubernetes.Interface) *K8sProvider {
	return &K8sProvider{client: client}
}

// GetFailureEvents returns pod failure events from the given namespace within the time window.
// If namespace is empty, events from all namespaces are returned.
func (p *K8sProvider) GetFailureEvents(ctx context.Context, namespace string, windowSeconds int) ([]PodEvent, error) {
	cutoff := time.Now().Add(-time.Duration(windowSeconds) * time.Second)

	eventList, err := p.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	var events []PodEvent
	for _, e := range eventList.Items {
		if !failureReasons[e.Reason] {
			continue
		}
		// Check event time (LastTimestamp or EventTime)
		eventTime := e.LastTimestamp.Time
		if eventTime.IsZero() {
			eventTime = e.EventTime.Time
		}
		if eventTime.Before(cutoff) {
			continue
		}
		events = append(events, PodEvent{
			Namespace: e.Namespace,
			Pod:       e.InvolvedObject.Name,
			Reason:    e.Reason,
			Message:   e.Message,
		})
	}

	return events, nil
}
