package aggregator

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregatorDiscoveryAndScrape(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	nodes := []NodeEndpoint{
		{Name: "node-1", Address: "10.0.0.1", Port: 8080},
		{Name: "node-2", Address: "10.0.0.2", Port: 8080},
	}

	metrics := map[string]*ScrapedPayload{
		"http://10.0.0.1:8080/metrics": {
			Node:      "node-1",
			Timestamp: time.Now(),
			Monitors:  []ScrapedMetric{{ID: "cpu", Type: "system", Status: "Green", Value: "50%"}},
		},
		"http://10.0.0.2:8080/metrics": {
			Node:      "node-2",
			Timestamp: time.Now(),
			Monitors:  []ScrapedMetric{{ID: "cpu", Type: "system", Status: "Amber", Value: "85%"}},
		},
	}

	provider := NewMockProvider(nodes, metrics)
	agg := NewAggregator(provider, "app=lmon-node", 8080, "/metrics", 100*time.Millisecond, nil)
	agg.Start(ctx)
	defer agg.Stop()

	assert.Eventually(t, func() bool {
		results := agg.Results()
		return len(results) == 2 && results["node-1"].Available && results["node-2"].Available
	}, time.Second, 10*time.Millisecond, "should discover and scrape both nodes")

	results := agg.Results()
	assert.Equal(t, "node-1", results["node-1"].Node)
	assert.NotNil(t, results["node-1"].Metrics)
	assert.Equal(t, 1, len(results["node-1"].Metrics.Monitors))
}

func TestAggregatorUnreachableNode(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	nodes := []NodeEndpoint{
		{Name: "node-1", Address: "10.0.0.1", Port: 8080},
	}

	provider := NewMockProvider(nodes, nil)
	provider.SetScrapeError("http://10.0.0.1:8080/metrics", fmt.Errorf("connection refused"))

	var pushCalled atomic.Int32
	push := func(_ string, _ bool, _ string) {
		pushCalled.Add(1)
	}

	agg := NewAggregator(provider, "app=lmon-node", 8080, "/metrics", 100*time.Millisecond, push)
	agg.Start(ctx)
	defer agg.Stop()

	assert.Eventually(t, func() bool {
		results := agg.Results()
		r, ok := results["node-1"]
		return ok && !r.Available && r.Error != ""
	}, time.Second, 10*time.Millisecond, "node should be marked unavailable")

	assert.Greater(t, pushCalled.Load(), int32(0), "push should be called for unavailable node")
}

func TestAggregatorWebhookOnStateChange(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	nodes := []NodeEndpoint{
		{Name: "node-1", Address: "10.0.0.1", Port: 8080},
	}

	provider := NewMockProvider(nodes, map[string]*ScrapedPayload{
		"http://10.0.0.1:8080/metrics": {Node: "node-1", Monitors: []ScrapedMetric{}},
	})

	var pushCount atomic.Int32
	push := func(_ string, _ bool, _ string) {
		pushCount.Add(1)
	}

	agg := NewAggregator(provider, "app=lmon-node", 8080, "/metrics", 50*time.Millisecond, push)
	agg.Start(ctx)

	// Wait for initial scrape (should be available, no push for initial available state)
	assert.Eventually(t, func() bool {
		results := agg.Results()
		r, ok := results["node-1"]
		return ok && r.Available
	}, time.Second, 10*time.Millisecond)

	// Make node fail
	provider.SetScrapeError("http://10.0.0.1:8080/metrics", fmt.Errorf("timeout"))

	// Wait for failure detection
	assert.Eventually(t, func() bool {
		results := agg.Results()
		r, ok := results["node-1"]
		return ok && !r.Available
	}, time.Second, 10*time.Millisecond, "node should become unavailable")

	// Recover the node
	provider.SetScrapeError("http://10.0.0.1:8080/metrics", nil)

	assert.Eventually(t, func() bool {
		results := agg.Results()
		r, ok := results["node-1"]
		return ok && r.Available
	}, time.Second, 10*time.Millisecond, "node should recover")

	agg.Stop()

	// Should have at least failure and recovery pushes
	assert.GreaterOrEqual(t, pushCount.Load(), int32(2), "should have push for failure and recovery")
}

func TestAggregatorNodeDisappears(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	nodes := []NodeEndpoint{
		{Name: "node-1", Address: "10.0.0.1", Port: 8080},
		{Name: "node-2", Address: "10.0.0.2", Port: 8080},
	}

	provider := NewMockProvider(nodes, map[string]*ScrapedPayload{
		"http://10.0.0.1:8080/metrics": {Node: "node-1"},
		"http://10.0.0.2:8080/metrics": {Node: "node-2"},
	})

	agg := NewAggregator(provider, "app=lmon-node", 8080, "/metrics", 50*time.Millisecond, nil)
	agg.Start(ctx)
	defer agg.Stop()

	// Wait for both nodes
	assert.Eventually(t, func() bool {
		results := agg.Results()
		return len(results) == 2
	}, time.Second, 10*time.Millisecond)

	// Remove node-2
	provider.SetNodes([]NodeEndpoint{
		{Name: "node-1", Address: "10.0.0.1", Port: 8080},
	})

	assert.Eventually(t, func() bool {
		results := agg.Results()
		r, ok := results["node-2"]
		return ok && !r.Available
	}, time.Second, 10*time.Millisecond, "disappeared node should be marked unavailable")
}

func TestAggregatorParallelScraping(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Create many nodes to verify parallel scraping
	nodes := make([]NodeEndpoint, 10)
	metrics := make(map[string]*ScrapedPayload)
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("node-%d", i)
		addr := fmt.Sprintf("10.0.0.%d", i+1)
		nodes[i] = NodeEndpoint{Name: name, Address: addr, Port: 8080}
		endpoint := fmt.Sprintf("http://%s:8080/metrics", addr)
		metrics[endpoint] = &ScrapedPayload{Node: name, Monitors: []ScrapedMetric{}}
	}

	provider := NewMockProvider(nodes, metrics)
	agg := NewAggregator(provider, "app=lmon-node", 8080, "/metrics", 100*time.Millisecond, nil)
	agg.Start(ctx)
	defer agg.Stop()

	assert.Eventually(t, func() bool {
		results := agg.Results()
		if len(results) != 10 {
			return false
		}
		for _, r := range results {
			if !r.Available {
				return false
			}
		}
		return true
	}, 2*time.Second, 10*time.Millisecond, "all 10 nodes should be scraped")
}

func TestAggregatorDefaults(t *testing.T) {
	provider := NewMockProvider(nil, nil)
	agg := NewAggregator(provider, "app=lmon", 0, "", 100*time.Millisecond, nil)
	require.Equal(t, 8080, agg.nodePort)
	require.Equal(t, "/metrics", agg.metricsPath)
}
