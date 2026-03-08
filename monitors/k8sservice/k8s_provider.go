package k8sservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sProvider discovers pod endpoints behind a Kubernetes service and probes their health.
type K8sProvider struct {
	client     kubernetes.Interface
	httpClient *http.Client
}

// NewK8sProvider creates a new K8sProvider with the given clientset.
func NewK8sProvider(client kubernetes.Interface) *K8sProvider {
	return &K8sProvider{
		client:     client,
		httpClient: &http.Client{},
	}
}

// GetPodHealth discovers pods behind a service and probes each pod's health endpoint.
func (p *K8sProvider) GetPodHealth(ctx context.Context, namespace, service string, port int, healthPath string, timeout int) ([]PodEndpoint, error) {
	// Get the endpoints for the service
	ep, err := p.client.CoreV1().Endpoints(namespace).Get(ctx, service, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get endpoints for %s/%s: %w", namespace, service, err)
	}

	var pods []PodEndpoint
	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			name := ""
			if addr.TargetRef != nil {
				name = addr.TargetRef.Name
			}
			pods = append(pods, PodEndpoint{
				Name: name,
				IP:   addr.IP,
			})
		}
	}

	// Probe each pod's health endpoint
	timeoutDuration := time.Duration(timeout) * time.Millisecond
	for i := range pods {
		url := fmt.Sprintf("http://%s:%d%s", pods[i].IP, port, healthPath)
		pods[i].Healthy = probeHealth(ctx, p.httpClient, url, timeoutDuration)
	}

	return pods, nil
}

// probeHealth makes an HTTP GET request and returns true if the response status is 2xx.
func probeHealth(ctx context.Context, client *http.Client, url string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
