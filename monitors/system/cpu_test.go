// cpu_test.go contains unit tests for the Cpu monitor implementation and its integration with the monitoring service.
package system

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	"lmon/config"
	"lmon/monitors"
)

// TestNewCpu verifies that a Cpu monitor can be created and added to a monitoring service.
func TestNewCpu(t *testing.T) {
	push := monitors.NewMockPush()
	cpu := NewCpu(90, "", MockCpuProvider{Current: atomic.NewFloat64(0)})
	svc := monitors.NewService(t.Context(), time.Second, time.Millisecond, push.Push)
	svc.Add(t.Context(), cpu)
	assert.Equal(t, 1, svc.Size(), "one monitor added")
}

// TestCpu_DisplayName verifies that DisplayName returns the expected string for the CPU monitor.
func TestCpu_DisplayName(t *testing.T) {
	c := Cpu{}
	assert.Equal(t, "cpu", c.DisplayName(), "DisplayName should return 'cpu'")
}

// TestCpu_Group verifies that Group returns the correct group name for the CPU monitor.
func TestCpu_Group(t *testing.T) {
	c := Cpu{}
	assert.Equal(t, Group, c.Group(), "Group should return the Group constant value")
	assert.Equal(t, "system", c.Group(), "Group should return 'system'")
}

// TestCpu_Name verifies that Name returns the correct unique identifier for the CPU monitor.
func TestCpu_Name(t *testing.T) {
	c := Cpu{}
	expected := Group + "_cpu"
	assert.Equal(t, expected, c.Name(), "Name should return Group+'_cpu'")
	assert.Equal(t, "system_cpu", c.Name(), "Name should return 'system_cpu'")
}

// TestCpu_Save verifies that Save correctly persists the CPU monitor configuration.
func TestCpu_Save(t *testing.T) {
	l := config.NewLoader("", []string{t.TempDir()})
	cfg, _ := l.Load()

	// Arrange
	c := Cpu{
		threshold: 42,
		icon:      "icon-test",
	}

	// Act
	c.Save(cfg)

	// Assert
	assert.Equal(t, 42, cfg.Monitoring.System.CPU.Threshold, "Save should set the threshold")
	assert.Equal(t, "icon-test", cfg.Monitoring.System.CPU.Icon, "Save should set the icon")
}

// TestCpu_DefaultImpl verifies that the defaultCpuProvider initializes and returns plausible values.
func TestCpu_DefaultImpl(t *testing.T) {
	impl := newDefaultCpuProvider()
	assert.NotEqual(t, 0, impl.lastCPUCheck)
	assert.Less(t, 0.0, impl.prevCPUTimes.System+impl.prevCPUTimes.User+impl.prevCPUTimes.Idle)
}

// TestCpu_DefaultImplSmokeTest verifies that the default implementation of Cpu.Check does not return an error status.
func TestCpu_DefaultImplSmokeTest(t *testing.T) {
	c := NewCpu(100, "", nil)
	r := c.Check(t.Context())
	assert.NotEqual(t, monitors.RAGError, r.Status, "status not error")
}

// TestCpu_Check_Mock verifies Cpu.Check with a mock provider for various usage scenarios and thresholds.
func TestCpu_Check_Mock(t *testing.T) {
	tests := []struct {
		name      string
		usage     float64
		err       error
		threshold int
		want      monitors.Result
	}{
		{"green 50", 50, nil, 90, monitors.Result{Status: monitors.RAGGreen, Value: "50.0%"}},
		{"green 90", 80, nil, 90, monitors.Result{Status: monitors.RAGGreen, Value: "80.0%"}},
		{"amber 81", 81, nil, 90, monitors.Result{Status: monitors.RAGAmber, Value: "81.0%"}},
		{"amber 89", 89, nil, 90, monitors.Result{Status: monitors.RAGAmber, Value: "89.0%"}},
		{"red 90", 90, nil, 90, monitors.Result{Status: monitors.RAGRed, Value: "90.0%"}},
		{"red 100", 100, nil, 90, monitors.Result{Status: monitors.RAGRed, Value: "100.0%"}},
		{"err", 100, os.ErrPermission, 90, monitors.Result{Status: monitors.RAGError, Value: "error getting CPU Current: permission denied"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCpu(tt.threshold, "", MockCpuProvider{Current: atomic.NewFloat64(tt.usage), err: tt.err})
			r := c.Check(t.Context())
			assert.Equal(t, tt.want.Status, r.Status, "status")
			assert.Equal(t, tt.want.Value, r.Value, "value")
		})

	}
}
