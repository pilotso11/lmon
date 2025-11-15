// builder_test.go contains unit tests for the Mapper type and its monitor construction methods.
package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"lmon/config"
	"lmon/monitors/ping"
)

// TestNewMapper verifies that NewMapper initializes the Impls field even if nil is passed.
func TestNewMapper(t *testing.T) {
	m := NewMapper(nil)
	assert.NotNil(t, m.Impls, "impl is not null")
}

// TestNewDisk verifies that NewDisk creates a disk monitor with the correct name and no error.
func TestNewDisk(t *testing.T) {
	m := NewMapper(nil)
	d, err := m.NewDisk(t.Context(), "test", config.DiskConfig{
		Threshold: 50,
		Icon:      "",
		Path:      "/tmp",
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "disk_test", d.Name())
}

// TestMapper_NewHealthcheck verifies that NewHealthcheck creates a healthcheck monitor with the correct name and no error.
func TestMapper_NewHealthcheck(t *testing.T) {
	m := NewMapper(nil)
	h, err := m.NewHealthcheck(t.Context(), "test", config.HealthcheckConfig{
		URL:     "http://localhost:8080",
		Timeout: 10,
		Icon:    "",
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "health_test", h.Name(), "should create healthcheck with correct name")
}

// TestMapper_NewPing verifies that NewPing creates a ping monitor with the correct name, provider, and error handling.
func TestMapper_NewPing(t *testing.T) {
	// Production (default provider)
	m := NewMapper(nil)
	p, err := m.NewPing(t.Context(), "pingtest", config.PingConfig{
		Address:        "127.0.0.1",
		Timeout:        1000,
		Icon:           "",
		AmberThreshold: 100,
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "ping_pingtest", p.Name(), "should create ping monitor with correct name")
	assert.Equal(t, "Ping: pingtest (127.0.0.1)", p.DisplayName(), "display name should match")

	// Test with mock provider
	mockProvider := ping.NewMockPingProvider(42, nil)
	m2 := NewMapper(&Implementations{Ping: mockProvider})
	p2, err := m2.NewPing(t.Context(), "mockping", config.PingConfig{
		Address:        "localhost",
		Timeout:        500,
		Icon:           "wifi",
		AmberThreshold: 50,
	})
	assert.NoError(t, err, "should not error with mock provider")
	assert.Equal(t, "ping_mockping", p2.Name(), "should create ping monitor with correct name")
	assert.Equal(t, "Ping: mockping (localhost)", p2.DisplayName(), "display name should match")

	// Error case: missing amberThreshold
	p, err = m.NewPing(t.Context(), "badping", config.PingConfig{
		Address: "localhost",
		Timeout: 500,
		Icon:    "wifi",
		// amberThreshold missing
	})
	assert.NoError(t, err, "should default not error if amberThreshold is missing or <= 0")
}

// TestMapper_NewCpu verifies that NewCpu creates a CPU monitor with the correct name and no error.
func TestMapper_NewCpu(t *testing.T) {
	m := NewMapper(nil)
	c, err := m.NewCpu(t.Context(), config.SystemItem{
		Threshold: 50,
		Icon:      "",
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "system_cpu", c.Name(), "should create cpu monitor with correct name")
}

// TestMapper_NewMem verifies that NewMem creates a memory monitor with the correct name and no error.
func TestMapper_NewMem(t *testing.T) {
	m := NewMapper(nil)
	mem, err := m.NewMem(t.Context(), config.SystemItem{
		Threshold: 50,
		Icon:      "",
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "system_mem", mem.Name(), "should create memory monitor with correct name")
}

// TestMapper_NewDocker verifies that NewDocker creates a Docker monitor with the correct name and no error.
func TestMapper_NewDocker(t *testing.T) {
	m := NewMapper(nil)
	d, err := m.NewDocker(t.Context(), "test", config.DockerConfig{
		Containers: "web-app, api-server",
		Threshold:  5,
		Icon:       "box",
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "docker_test", d.Name(), "should create docker monitor with correct name")
}
