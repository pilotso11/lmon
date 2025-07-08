// disk_test.go contains unit tests for the Disk monitor implementation and its integration with the monitoring service.
package disk

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"lmon/config"
	"lmon/monitors"
)

// TstNewDisk is a helper for quickly testing Disk creation and addition to a monitoring service.
func TstNewDisk(t *testing.T) {
	push := monitors.NewMockPush()
	d := NewDisk("test", "/test", 90, "", MockDiskProvider{path: "/test", Current: atomic.NewFloat64(0), total: 100})
	svc := monitors.NewService(t.Context(), time.Second, time.Millisecond, push.Push)
	_ = svc.Add(t.Context(), d)
	assert.Equal(t, 1, svc.Size(), "one monitor added")
}

// TestDisk_DisplayName verifies that DisplayName returns the expected string for various disk names and paths.
func TestDisk_DisplayName(t *testing.T) {
	tests := []struct {
		name   string
		fields Disk
		want   string
	}{
		{"/", Disk{name: "root", path: "/"}, "root (/)"},
		{"/home", Disk{name: "homes", path: "/home"}, "homes (/home)"},
		{"/mnt/remote/vol1", Disk{name: "rmt1", path: "/mnt/remote/vol1"}, "rmt1 (/mnt/remote/vol1)"},
		// Edge cases
		{"empty name and path", Disk{name: "", path: ""}, " ()"},
		{"empty name", Disk{name: "", path: "/data"}, " (/data)"},
		{"empty path", Disk{name: "data", path: ""}, "data ()"},
		{"long name and path", Disk{name: "verylongdiskname", path: "/this/is/a/very/long/path/for/testing"}, "verylongdiskname (/this/is/a/very/long/path/for/testing)"},
		{"special chars", Disk{name: "spécial", path: "/päth/with/üñîçødë"}, "spécial (/päth/with/üñîçødë)"},
		{"whitespace", Disk{name: " ", path: " "}, "  ( )"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDisk(tt.fields.name, tt.fields.path, tt.fields.threshold, "", tt.fields.impl)
			assert.Equalf(t, tt.want, d.DisplayName(), "DisplayName()")
			assert.Equal(t, Icon, d.icon, "icon")
		})
	}
}

// TestDisk_Group verifies that Group returns the correct group name for disk monitors.
func TestDisk_Group(t *testing.T) {
	tests := []struct {
		name   string
		fields Disk
		want   string
	}{
		{"basic", Disk{name: "root", path: "/"}, Group},
		{"empty", Disk{}, Group},
		{"custom", Disk{name: "data", path: "/data"}, Group},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Disk{
				name:      tt.fields.name,
				path:      tt.fields.path,
				threshold: tt.fields.threshold,
				icon:      tt.fields.icon,
				impl:      tt.fields.impl,
			}
			assert.Equalf(t, tt.want, d.Group(), "Group()")
		})
	}
}

// TestDisk_Name verifies that Name returns the correct unique identifier for disk monitors.
func TestDisk_Name(t *testing.T) {
	tests := []struct {
		name   string
		fields Disk
		want   string
	}{
		{"basic", Disk{name: "root", path: "/"}, "disk_root"},
		{"empty", Disk{}, "disk_"},
		{"custom", Disk{name: "data", path: "/data"}, "disk_data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Disk{
				name:      tt.fields.name,
				path:      tt.fields.path,
				threshold: tt.fields.threshold,
				icon:      tt.fields.icon,
				impl:      tt.fields.impl,
			}
			assert.Equalf(t, tt.want, d.Name(), "Name()")
		})
	}
}

// TestDisk_Save verifies that Save correctly persists the disk monitor configuration.
func TestDisk_Save(t *testing.T) {
	l := config.NewLoader("", []string{t.TempDir()})
	cfg, _ := l.Load()

	d := NewDisk("test", "/test", 66, "", nil)
	d.Save(cfg)
	assert.Equal(t, 1, len(cfg.Monitoring.Disk), "len disk")
	dc, ok := cfg.Monitoring.Disk["test"]
	require.True(t, ok, "disk config exists")
	assert.Equal(t, "/test", dc.Path)
	assert.Equal(t, Icon, dc.Icon)
	assert.Equal(t, 66, dc.Threshold)
}

// TestDisk_Check_DefaultImpl verifies that the default implementation of Disk.Check does not panic and returns a valid status.
func TestDisk_Check_DefaultImpl(t *testing.T) {
	assert.NotPanics(t, func() {
		d := NewDisk("local", t.TempDir(), 90, "", nil)
		res := d.Check(t.Context())
		assert.NotEqual(t, monitors.RAGError, res.Status, "status not error")
	})
}

// TestDisk_Check_Mock verifies Disk.Check with a mock provider for various usage scenarios and thresholds.
func TestDisk_Check_Mock(t *testing.T) {
	tests := []struct {
		name      string
		usage     float64
		total     float64
		err       error
		threshold int
		want      monitors.Result
	}{
		{"green 50", 50, 100 * gigabyte, nil, 90, monitors.Result{Status: monitors.RAGGreen, Value: "50.0% used (50.0 GB / 100.0 GB)"}},
		{"green 90", 80, 100 * gigabyte, nil, 90, monitors.Result{Status: monitors.RAGGreen, Value: "80.0% used (80.0 GB / 100.0 GB)"}},
		{"amber 81", 81, 100 * gigabyte, nil, 90, monitors.Result{Status: monitors.RAGAmber, Value: "81.0% used (81.0 GB / 100.0 GB)"}},
		{"amber 89", 89, 100 * gigabyte, nil, 90, monitors.Result{Status: monitors.RAGAmber, Value: "89.0% used (89.0 GB / 100.0 GB)"}},
		{"red 90", 90, 100 * gigabyte, nil, 90, monitors.Result{Status: monitors.RAGRed, Value: "90.0% used (90.0 GB / 100.0 GB)"}},
		{"red 100", 100, 100 * gigabyte, nil, 90, monitors.Result{Status: monitors.RAGRed, Value: "100.0% used (100.0 GB / 100.0 GB)"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDisk("test", "/test", tt.threshold, "", MockDiskProvider{Current: atomic.NewFloat64(tt.usage), total: tt.total, err: tt.err, path: "/test"})
			r := d.Check(t.Context())
			assert.Equal(t, tt.want.Status, r.Status, "status")
			assert.Equal(t, tt.want.Value, r.Value, "value")
		})

	}
}

// Test_Check_PushOnAddWithBreach verifies that adding a disk monitor with a breached threshold triggers a push notification.
func Test_Check_PushOnAddWithBreach(t *testing.T) {
	push := monitors.NewMockPush()

	svc := monitors.NewService(t.Context(), 10*time.Millisecond, time.Millisecond, push.Push)
	d := NewDisk("test", "/test", 90, "", MockDiskProvider{Current: atomic.NewFloat64(90), total: 100 * gigabyte, err: nil, path: "/test"})
	err := svc.Add(t.Context(), d)
	assert.NoError(t, err)

	require.Equal(t, 1, push.Calls.Size(), "push on add for breach is expected")
	val, ok := push.Calls.Load(int32(1))
	assert.True(t, ok, "push exists")
	assert.Equal(t, monitors.RAGRed, val.Result.Status, "status")
}

// Test_Check_PushOnAddWithErr verifies that adding a disk monitor with an error triggers a push notification and returns an error.
func Test_Check_PushOnAddWithErr(t *testing.T) {
	push := monitors.NewMockPush()

	svc := monitors.NewService(t.Context(), 10*time.Millisecond, time.Millisecond, push.Push)
	d := NewDisk("test", "/test", 90, "", MockDiskProvider{Current: atomic.NewFloat64(0), total: 100 * gigabyte, err: os.ErrNotExist, path: "/test"})

	err := svc.Add(t.Context(), d)
	assert.NoError(t, err, "error not expected even with failed check on add")

	require.Equal(t, 1, push.Calls.Size(), "push on add for breach is expected")
	val, ok := push.Calls.Load(int32(1))
	assert.True(t, ok, "push exists")
	assert.Equal(t, monitors.RAGError, val.Result.Status, "status")
}

// Test_Check_PushOnChange verifies that push notifications are triggered when the disk usage status changes.
func Test_Check_PushOnChange(t *testing.T) {
	tests := []struct {
		name          string
		initial       float64
		initialStatus monitors.RAG
		initialCnt    int
		second        float64
		secondStatus  monitors.RAG
		secondCnt     int
	}{
		{"green to red", 79, monitors.RAGGreen, 0, 90, monitors.RAGRed, 1},
		{"green to amber", 79, monitors.RAGGreen, 0, 81, monitors.RAGAmber, 1},
		{"amber to red", 81, monitors.RAGAmber, 1, 90, monitors.RAGRed, 2},
		{"red to amber", 90, monitors.RAGRed, 1, 89, monitors.RAGAmber, 2},
		{"amber to green", 81, monitors.RAGAmber, 1, 80, monitors.RAGGreen, 2},
		{"red to green", 90, monitors.RAGRed, 1, 80, monitors.RAGGreen, 2},
		{"green to green", 50, monitors.RAGGreen, 0, 60, monitors.RAGGreen, 0},
		{"amber to amber", 85, monitors.RAGAmber, 1, 86, monitors.RAGAmber, 1},
		{"red to red", 92, monitors.RAGRed, 1, 95, monitors.RAGRed, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			push := monitors.NewMockPush()
			svc := monitors.NewService(t.Context(), 10*time.Millisecond, time.Millisecond, push.Push)
			impl := MockDiskProvider{Current: atomic.NewFloat64(tt.initial), total: 100 * gigabyte, path: "/test"}
			d := NewDisk("test", "/test", 90, "", &impl)
			err := svc.Add(t.Context(), d)
			assert.NoError(t, err, "error not expected")

			require.Equal(t, tt.initialCnt, push.Calls.Size(), "initial push cnt")
			if tt.initialCnt > 0 {
				va, ok := push.Calls.Load(int32(tt.initialCnt))
				assert.True(t, ok, "push exists")
				assert.Equal(t, tt.initialStatus, va.Result.Status, "initial status")
			}

			// toggle usage to new value
			impl.Current.Store(tt.second)

			time.Sleep(15 * time.Millisecond)
			require.Equal(t, tt.secondCnt, push.Calls.Size(), "push cnt after toggle")
			if tt.secondCnt > 0 {
				va, ok := push.Calls.Load(int32(tt.secondCnt))
				assert.True(t, ok, "push exists")
				assert.Equal(t, tt.secondStatus, va.Result.Status, "initial status")
			}
		})
	}

}
