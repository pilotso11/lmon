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

func TestNewMem(t *testing.T) {
	push := monitors.NewMockPush()
	d := NewMem(90, "", MockMemProvider{Current: atomic.NewFloat64(0)})
	svc := monitors.NewService(t.Context(), time.Second, time.Millisecond, push.Push)
	_ = svc.Add(t.Context(), d)
	assert.Equal(t, 1, svc.Size(), "one monitor added")
}

func TestMem_DisplayName(t *testing.T) {
	c := Mem{}
	assert.Equal(t, "mem", c.DisplayName(), "DisplayName should return 'mem'")
}

func TestMem_Group(t *testing.T) {
	c := Mem{}
	assert.Equal(t, Group, c.Group(), "Group should return the Group constant value")
	assert.Equal(t, "system", c.Group(), "Group should return 'system'")
}

func TestMem_Name(t *testing.T) {
	c := Mem{}
	expected := Group + "_mem"
	assert.Equal(t, expected, c.Name(), "Name should return Group+'_mem'")
	assert.Equal(t, "system_mem", c.Name(), "Name should return 'system_mem'")
}

func TestMem_Save(t *testing.T) {
	l := config.NewLoader("", []string{t.TempDir()})
	cfg, _ := l.Load()

	// Arrange
	c := Mem{
		threshold: 42,
		icon:      "icon-test",
	}

	// Act
	c.Save(cfg)

	// Assert
	assert.Equal(t, 42, cfg.Monitoring.System.Memory.Threshold, "Save should set the threshold")
	assert.Equal(t, "icon-test", cfg.Monitoring.System.Memory.Icon, "Save should set the icon")
}

func TestMem_DefaultImplSmokeTest(t *testing.T) {
	c := NewMem(100, "", nil)
	r := c.Check(t.Context())
	assert.NotEqual(t, monitors.RAGError, r.Status, "status not error")
	assert.NotEqual(t, "0.0%", r.Value)
}

func TestMem_Check_Mock(t *testing.T) {
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
		{"err", 100, os.ErrPermission, 90, monitors.Result{Status: monitors.RAGError, Value: "error getting mem Current: permission denied"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMem(tt.threshold, "", MockMemProvider{Current: atomic.NewFloat64(tt.usage), err: tt.err})
			r := c.Check(t.Context())
			assert.Equal(t, tt.want.Status, r.Status, "status")
			assert.Equal(t, tt.want.Value, r.Value, "value")
		})

	}
}
