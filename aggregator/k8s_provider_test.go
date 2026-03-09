package aggregator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubPodLister implements PodLister for testing K8sProvider delegation.
type stubPodLister struct {
	endpoints []NodeEndpoint
	err       error
}

func (s *stubPodLister) ListPods(_ context.Context, _ string) ([]NodeEndpoint, error) {
	return s.endpoints, s.err
}

func TestK8sProvider_DiscoverNodes_DelegatesToPodLister(t *testing.T) {
	expected := []NodeEndpoint{
		{Name: "node-1", Address: "10.0.0.1"},
		{Name: "node-2", Address: "10.0.0.2"},
	}
	lister := &stubPodLister{endpoints: expected}

	provider := NewK8sProvider(lister)
	nodes, err := provider.DiscoverNodes(t.Context(), "app=lmon")
	require.NoError(t, err)
	assert.Equal(t, expected, nodes)
}

func TestK8sProvider_DiscoverNodes_PropagatesError(t *testing.T) {
	lister := &stubPodLister{err: assert.AnError}

	provider := NewK8sProvider(lister)
	_, err := provider.DiscoverNodes(t.Context(), "app=lmon")
	require.ErrorIs(t, err, assert.AnError)
}

func TestK8sProvider_ScrapeMetrics_Success(t *testing.T) {
	payload := ScrapedPayload{
		Node:      "node-1",
		Timestamp: time.Now().Truncate(time.Second),
		Monitors: []ScrapedMetric{
			{ID: "cpu", Type: "system", Status: "Green", Value: "50%"},
			{ID: "mem", Type: "system", Status: "Amber", Value: "85%"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	provider := NewK8sProvider(&stubPodLister{})
	provider.httpClient = server.Client()

	result, err := provider.ScrapeMetrics(t.Context(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, "node-1", result.Node)
	assert.Len(t, result.Monitors, 2)
	assert.Equal(t, "cpu", result.Monitors[0].ID)
	assert.Equal(t, "Green", result.Monitors[0].Status)
}

func TestK8sProvider_ScrapeMetrics_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := NewK8sProvider(&stubPodLister{})
	provider.httpClient = server.Client()

	_, err := provider.ScrapeMetrics(t.Context(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestK8sProvider_ScrapeMetrics_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not json at all"))
	}))
	defer server.Close()

	provider := NewK8sProvider(&stubPodLister{})
	provider.httpClient = server.Client()

	_, err := provider.ScrapeMetrics(t.Context(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestK8sProvider_ScrapeMetrics_ConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	server.Close() // close immediately to force connection error

	provider := NewK8sProvider(&stubPodLister{})

	_, err := provider.ScrapeMetrics(t.Context(), server.URL)
	require.Error(t, err)
}
