package web

import (
	"net/http"

	"go.uber.org/atomic"

	"lmon/monitors/disk"
	"lmon/monitors/docker"
	"lmon/monitors/healthcheck"
	"lmon/monitors/k8sevents"
	"lmon/monitors/k8snodes"
	"lmon/monitors/k8sservice"
	"lmon/monitors/mapper"
	"lmon/monitors/ping"
	"lmon/monitors/system"
)

// MockWebhookHandler is a test double for capturing webhook callback invocations.
type MockWebhookHandler struct {
	LastMessage atomic.String
	Cnt         atomic.Int32
}

// webhookCallback stores the last message and increments the call count.
func (m *MockWebhookHandler) webhookCallback(msg string) {
	m.LastMessage.Store(msg)
	m.Cnt.Inc()
}

// NewMockImplementations returns a set of mock monitor providers and a webhook callback for testing.
func NewMockImplementations(hook *MockWebhookHandler) *mapper.Implementations {
	return &mapper.Implementations{
		Disk:       disk.NewMockDiskProvider(50),                          // percent usage
		Health:     healthcheck.NewMockHealthcheckProvider(http.StatusOK), // http status code
		Cpu:        system.NewMockCpuProvider(50),                         // percent usage
		Mem:        system.NewMockMemProvider(50),                         // percent usage
		Ping:       ping.NewMockPingProvider(50, nil),                     // default mock ping provider
		Docker:     docker.NewMockDockerProvider(),
		K8sEvents:  k8sevents.NewMockProvider(nil, nil),
		K8sNodes:   k8snodes.NewMockProvider(nil, nil),
		K8sService: k8sservice.NewMockProvider(nil, nil),
		Webhook:    hook.webhookCallback,
	}
}
