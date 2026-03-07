package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// K8sProvider discovers lmon node pods via Kubernetes API and scrapes their metrics.
type K8sProvider struct {
	httpClient *http.Client
	podLister  PodLister
}

// PodLister abstracts Kubernetes pod listing for testability.
type PodLister interface {
	ListPods(ctx context.Context, label string) ([]NodeEndpoint, error)
}

// NewK8sProvider creates a new K8sProvider.
func NewK8sProvider(lister PodLister) *K8sProvider {
	return &K8sProvider{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		podLister: lister,
	}
}

func (p *K8sProvider) DiscoverNodes(ctx context.Context, label string) ([]NodeEndpoint, error) {
	return p.podLister.ListPods(ctx, label)
}

func (p *K8sProvider) ScrapeMetrics(ctx context.Context, endpoint string) (*ScrapedPayload, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scrape %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scrape %s: status %d", endpoint, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response %s: %w", endpoint, err)
	}

	var payload ScrapedPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode response %s: %w", endpoint, err)
	}

	return &payload, nil
}
