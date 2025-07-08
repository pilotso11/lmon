package web

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
	"lmon/monitors/system"
)

// TestMockWebhookHandler verifies that mockWebhookHandler correctly stores the last message and increments the count.
func TestMockWebhookHandler(t *testing.T) {
	h := &MockWebhookHandler{}
	assert.Equal(t, int32(0), h.Cnt.Load(), "Initial count should be 0")
	assert.Equal(t, "", h.LastMessage.Load(), "Initial message should be empty")

	h.webhookCallback("test message 1")
	assert.Equal(t, int32(1), h.Cnt.Load(), "Count should increment after first callback")
	assert.Equal(t, "test message 1", h.LastMessage.Load(), "Last message should be updated")

	h.webhookCallback("test message 2")
	assert.Equal(t, int32(2), h.Cnt.Load(), "Count should increment after second callback")
	assert.Equal(t, "test message 2", h.LastMessage.Load(), "Last message should be updated again")
}

// TestNewMockImplementations verifies that NewMockImplementations returns non-nil mock providers and sets the webhook.
func TestNewMockImplementations(t *testing.T) {
	h := &MockWebhookHandler{}
	impls := NewMockImplementations(h)

	assert.NotNil(t, impls, "Implementations should not be nil")
	assert.NotNil(t, impls.Disk, "Disk provider should not be nil")
	assert.NotNil(t, impls.Health, "Health provider should not be nil")
	assert.NotNil(t, impls.Cpu, "CPU provider should not be nil")
	assert.NotNil(t, impls.Mem, "Mem provider should not be nil")
	assert.NotNil(t, impls.Webhook, "Webhook callback should not be nil")

	// Test that the webhook callback is the handler's method
	impls.Webhook("webhook test")
	assert.Equal(t, int32(1), h.Cnt.Load(), "Webhook callback should increment count")
	assert.Equal(t, "webhook test", h.LastMessage.Load(), "Webhook callback should store last message")
}

// TestNewMockImplementations_Integration ensures the returned implementations conform to expected interfaces.
func TestNewMockImplementations_Integration(t *testing.T) {
	h := &MockWebhookHandler{}
	impls := NewMockImplementations(h)

	// Disk provider should implement UsageProvider interface
	_, ok := interface{}(impls.Disk).(disk.UsageProvider)
	assert.True(t, ok, "Disk provider should implement UsageProvider")

	// Health provider should implement UsageProvider interface
	_, ok = interface{}(impls.Health).(healthcheck.UsageProvider)
	assert.True(t, ok, "Health provider should implement UsageProvider")

	// CPU provider should implement Usage() (float64, error)
	_, ok = interface{}(impls.Cpu).(system.CpuProvider)
	assert.True(t, ok, "CPU provider should implement Usage() (float64, error)")

	// Mem provider should implement Usage() (interface{}, error)
	_, ok = interface{}(impls.Mem).(system.MemProvider)
	assert.True(t, ok, "Mem provider should implement Usage() (interface{}, error)")
}
