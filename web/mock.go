package web

import (
	"net/http"

	"go.uber.org/atomic"

	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
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
		Disk:    disk.NewMockDiskProvider(50),                          // percent usage
		Health:  healthcheck.NewMockHealthcheckProvider(http.StatusOK), // http status code
		Cpu:     system.NewMockCpuProvider(50),                         // percent usage
		Mem:     system.NewMockMemProvider(50),                         // percent usage
		Ping:    &ping.MockPingProvider{ResponseMs: 50, Err: nil},      // default mock ping provider
		Webhook: hook.webhookCallback,
	}
}
