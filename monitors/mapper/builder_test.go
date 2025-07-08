// builder_test.go contains unit tests for the Mapper type and its monitor construction methods.
package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"lmon/config"
)

// TestNewMapper verifies that NewMapper initializes the Impls field even if nil is passed.
func TestNewMapper(t *testing.T) {
	m := NewMapper(nil)
	assert.NotNil(t, m.Impls, "impl is not null")
}

// TestNewDisk verifies that NewDisk creates a disk monitor with the correct name and no error.
func TestNewDisk(t *testing.T) {
	m := NewMapper(nil)
	d, err := m.NewDisk(nil, "test", config.DiskConfig{
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
	h, err := m.NewHealthcheck(nil, "test", config.HealthcheckConfig{
		URL:     "http://localhost:8080",
		Timeout: 10,
		Icon:    "",
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "health_test", h.Name(), "should create healthcheck with correct name")
}

// TestMapper_NewCpu verifies that NewCpu creates a CPU monitor with the correct name and no error.
func TestMapper_NewCpu(t *testing.T) {
	m := NewMapper(nil)
	c, err := m.NewCpu(nil, config.SystemItem{
		Threshold: 50,
		Icon:      "",
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "system_cpu", c.Name(), "should create cpu monitor with correct name")
}

// TestMapper_NewMem verifies that NewMem creates a memory monitor with the correct name and no error.
func TestMapper_NewMem(t *testing.T) {
	m := NewMapper(nil)
	mem, err := m.NewMem(nil, config.SystemItem{
		Threshold: 50,
		Icon:      "",
	})
	assert.NoError(t, err, "should not error")
	assert.Equal(t, "system_mem", mem.Name(), "should create memory monitor with correct name")
}
