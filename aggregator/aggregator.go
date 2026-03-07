// Package aggregator provides cluster-wide monitoring by discovering and scraping
// lmon node agents running as a DaemonSet in Kubernetes.
package aggregator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
)

// NodeEndpoint represents a discovered lmon node agent.
type NodeEndpoint struct {
	Name    string
	Address string
	Port    int
}

// ScrapedMetric represents a single monitor metric scraped from a node.
type ScrapedMetric struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// ScrapedPayload is the result of scraping a node's /metrics endpoint.
type ScrapedPayload struct {
	Node      string          `json:"node"`
	Timestamp time.Time       `json:"timestamp"`
	Monitors  []ScrapedMetric `json:"monitors"`
}

// NodeResult holds the scraped metrics from a single node.
type NodeResult struct {
	Node      string
	Timestamp time.Time
	Metrics   *ScrapedPayload
	Error     string
	Available bool
}

// Provider abstracts node discovery and metrics scraping for testability.
type Provider interface {
	DiscoverNodes(ctx context.Context, label string) ([]NodeEndpoint, error)
	ScrapeMetrics(ctx context.Context, endpoint string) (*ScrapedPayload, error)
}

// PushFunc is called when a node's availability state changes.
type PushFunc func(node string, available bool, message string)

// Aggregator discovers lmon nodes and scrapes their /metrics endpoints.
type Aggregator struct {
	results        *xsync.Map[string, NodeResult]
	provider       Provider
	nodeLabel      string
	nodePort       int
	metricsPath    string
	scrapeInterval time.Duration
	push           PushFunc
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	started        bool
}

// NewAggregator creates a new Aggregator with the given configuration.
func NewAggregator(provider Provider, nodeLabel string, nodePort int, metricsPath string, scrapeInterval time.Duration, push PushFunc) *Aggregator {
	if metricsPath == "" {
		metricsPath = "/metrics"
	}
	if nodePort == 0 {
		nodePort = 8080
	}
	return &Aggregator{
		results:        xsync.NewMap[string, NodeResult](),
		provider:       provider,
		nodeLabel:      nodeLabel,
		nodePort:       nodePort,
		metricsPath:    metricsPath,
		scrapeInterval: scrapeInterval,
		push:           push,
	}
}

// Start begins the aggregator's discovery and scrape loop.
// Safe to call only once; subsequent calls are no-ops.
func (a *Aggregator) Start(ctx context.Context) {
	if a.started {
		return
	}
	a.started = true
	ctx, a.cancel = context.WithCancel(ctx)
	a.wg.Add(1)
	go a.loop(ctx)
}

// Stop stops the aggregator and waits for the loop to finish.
func (a *Aggregator) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
}

// Results returns a snapshot of all node results.
func (a *Aggregator) Results() map[string]NodeResult {
	return xsync.ToPlainMap(a.results)
}

func (a *Aggregator) loop(ctx context.Context) {
	defer a.wg.Done()

	// Initial scrape
	a.scrapeAll(ctx)

	ticker := time.NewTicker(a.scrapeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.scrapeAll(ctx)
		}
	}
}

func (a *Aggregator) scrapeAll(ctx context.Context) {
	nodes, err := a.provider.DiscoverNodes(ctx, a.nodeLabel)
	if err != nil {
		log.Printf("Aggregator: discovery error: %v", err)
		return
	}

	// Scrape all nodes in parallel
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(n NodeEndpoint) {
			defer wg.Done()
			a.scrapeNode(ctx, n)
		}(node)
	}
	wg.Wait()

	// Mark nodes that disappeared as unavailable
	a.results.Range(func(name string, result NodeResult) bool {
		found := false
		for _, n := range nodes {
			if n.Name == name {
				found = true
				break
			}
		}
		if !found {
			if result.Available {
				a.results.Store(name, NodeResult{
					Node:      name,
					Timestamp: time.Now(),
					Available: false,
					Error:     "node no longer discovered",
				})
				if a.push != nil {
					a.push(name, false, "node no longer discovered")
				}
			}
		}
		return true
	})
}

func (a *Aggregator) scrapeNode(ctx context.Context, node NodeEndpoint) {
	endpoint := fmt.Sprintf("http://%s:%d%s", node.Address, a.nodePort, a.metricsPath)
	metrics, err := a.provider.ScrapeMetrics(ctx, endpoint)

	prev, hasPrev := a.results.Load(node.Name)

	if err != nil {
		result := NodeResult{
			Node:      node.Name,
			Timestamp: time.Now(),
			Available: false,
			Error:     err.Error(),
		}
		a.results.Store(node.Name, result)

		// Fire webhook if state changed
		if a.push != nil && (!hasPrev || prev.Available) {
			a.push(node.Name, false, err.Error())
		}
		return
	}

	result := NodeResult{
		Node:      node.Name,
		Timestamp: time.Now(),
		Metrics:   metrics,
		Available: true,
	}
	a.results.Store(node.Name, result)

	// Fire webhook on recovery
	if a.push != nil && hasPrev && !prev.Available {
		a.push(node.Name, true, "node recovered")
	}
}
