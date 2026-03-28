package monitors_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/docker"
	"lmon/monitors/healthcheck"
	"lmon/monitors/ping"
	"lmon/monitors/system"
)

// TestService_Save verifies that Service.Save correctly persists all monitor types to config.
func TestService_Save(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := monitors.NewService(ctx, time.Second, time.Second, nil)

	// Add a Disk monitor with default alertThreshold (0, which should become 1)
	diskMon := disk.NewDisk("disk1", "/mnt/disk1", 75, "hdd", 0, nil)
	svc.Add(ctx, diskMon)

	// Add a Healthcheck monitor with explicit alertThreshold
	healthMon, err := healthcheck.NewHealthcheck("health1", "http://localhost/health", 5, 401, "activity", "", 3, nil, nil, nil)
	require.NoError(t, err)
	svc.Add(ctx, healthMon)

	// Add a Ping monitor
	pingMon := ping.NewPingMonitor("ping1", "127.0.0.1", 1000, "wifi", 100, 0, nil)
	svc.Add(ctx, pingMon)

	cfg := &config.Config{}
	err = svc.Save(cfg)
	require.NoError(t, err, "service save should not error")

	// Check that each monitor type is present in config
	diskCfg, diskOk := cfg.Monitoring.Disk["disk1"]
	healthCfg, healthOk := cfg.Monitoring.Healthcheck["health1"]
	pingCfg, pingOk := cfg.Monitoring.Ping["ping1"]

	assert.True(t, diskOk, "disk monitor should be saved")
	assert.True(t, healthOk, "healthcheck monitor should be saved")
	assert.True(t, pingOk, "ping monitor should be saved")

	// Check values including alertthreshold
	assert.Equal(t, "/mnt/disk1", diskCfg.Path)
	assert.Equal(t, "hdd", diskCfg.Icon)
	assert.Equal(t, 1, diskCfg.AlertThreshold, "disk alertthreshold should default to 1")
	
	assert.Equal(t, "http://localhost/health", healthCfg.URL)
	assert.Equal(t, 401, healthCfg.RespCode)
	assert.Equal(t, "activity", healthCfg.Icon)
	assert.Equal(t, 3, healthCfg.AlertThreshold, "healthcheck alertthreshold should be 3")
	
	assert.Equal(t, "127.0.0.1", pingCfg.Address)
	assert.Equal(t, "wifi", pingCfg.Icon)
	assert.Equal(t, 100, pingCfg.AmberThreshold)
	assert.Equal(t, 1, pingCfg.AlertThreshold, "ping alertthreshold should default to 1")
}

// testMaintenance is a shared helper that creates a standard always-active MaintenanceConfig.
func testMaintenance() *config.MaintenanceConfig {
	return &config.MaintenanceConfig{
		Cron:     "* * * * *",
		Duration: 60,
	}
}

// TestService_Save_DiskMaintenance verifies that maintenance config is applied to disk entries after Save.
func TestService_Save_DiskMaintenance(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := monitors.NewService(ctx, time.Second, time.Second, nil)

	diskMon := disk.NewDisk("root", "/tmp", 80, "hdd", 1, nil)
	mc := testMaintenance()
	svc.AddWithMaintenance(ctx, diskMon, mc)

	cfg := &config.Config{}
	err := svc.Save(cfg)
	require.NoError(t, err)

	diskCfg, ok := cfg.Monitoring.Disk["root"]
	require.True(t, ok, "disk monitor should be saved to config")
	assert.Equal(t, mc.Cron, diskCfg.Maintenance.Cron, "disk maintenance cron should be saved")
	assert.Equal(t, mc.Duration, diskCfg.Maintenance.Duration, "disk maintenance duration should be saved")
}

// TestService_Save_HealthcheckMaintenance verifies that maintenance config is applied to healthcheck entries after Save.
func TestService_Save_HealthcheckMaintenance(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := monitors.NewService(ctx, time.Second, time.Second, nil)

	mockProvider := healthcheck.NewMockHealthcheckProvider(200)
	healthMon, err := healthcheck.NewHealthcheck("api", "http://localhost/health", 5, 200, "activity", "", 1, nil, mockProvider, nil)
	require.NoError(t, err)
	mc := testMaintenance()
	svc.AddWithMaintenance(ctx, healthMon, mc)

	cfg := &config.Config{}
	err = svc.Save(cfg)
	require.NoError(t, err)

	healthCfg, ok := cfg.Monitoring.Healthcheck["api"]
	require.True(t, ok, "healthcheck monitor should be saved to config")
	assert.Equal(t, mc.Cron, healthCfg.Maintenance.Cron, "healthcheck maintenance cron should be saved")
	assert.Equal(t, mc.Duration, healthCfg.Maintenance.Duration, "healthcheck maintenance duration should be saved")
}

// TestService_Save_PingMaintenance verifies that maintenance config is applied to ping entries after Save.
func TestService_Save_PingMaintenance(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := monitors.NewService(ctx, time.Second, time.Second, nil)

	mockPing := ping.NewMockPingProvider(10, nil)
	pingMon := ping.NewPingMonitor("gateway", "192.168.1.1", 1000, "wifi", 100, 1, mockPing)
	mc := testMaintenance()
	svc.AddWithMaintenance(ctx, pingMon, mc)

	cfg := &config.Config{}
	err := svc.Save(cfg)
	require.NoError(t, err)

	pingCfg, ok := cfg.Monitoring.Ping["gateway"]
	require.True(t, ok, "ping monitor should be saved to config")
	assert.Equal(t, mc.Cron, pingCfg.Maintenance.Cron, "ping maintenance cron should be saved")
	assert.Equal(t, mc.Duration, pingCfg.Maintenance.Duration, "ping maintenance duration should be saved")
}

// TestService_Save_DockerMaintenance verifies that maintenance config is applied to docker entries after Save.
func TestService_Save_DockerMaintenance(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := monitors.NewService(ctx, time.Second, time.Second, nil)

	mockDocker := docker.NewMockDockerProvider()
	dockerMon, err := docker.NewMonitor("webapp", "webapp-container", 5, "box", 1, nil, mockDocker)
	require.NoError(t, err)
	mc := testMaintenance()
	svc.AddWithMaintenance(ctx, dockerMon, mc)

	cfg := &config.Config{}
	err = svc.Save(cfg)
	require.NoError(t, err)

	dockerCfg, ok := cfg.Monitoring.Docker["webapp"]
	require.True(t, ok, "docker monitor should be saved to config")
	assert.Equal(t, mc.Cron, dockerCfg.Maintenance.Cron, "docker maintenance cron should be saved")
	assert.Equal(t, mc.Duration, dockerCfg.Maintenance.Duration, "docker maintenance duration should be saved")
}

// TestService_Save_SystemCPUMaintenance verifies that maintenance config is applied to the system CPU config after Save.
func TestService_Save_SystemCPUMaintenance(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := monitors.NewService(ctx, time.Second, time.Second, nil)

	mockCPU := system.NewMockCpuProvider(10)
	cpuMon := system.NewCpu(90, "cpu", 1, mockCPU)
	mc := testMaintenance()
	svc.AddWithMaintenance(ctx, cpuMon, mc)

	cfg := &config.Config{}
	err := svc.Save(cfg)
	require.NoError(t, err)

	assert.Equal(t, mc.Cron, cfg.Monitoring.System.CPU.Maintenance.Cron, "CPU maintenance cron should be saved")
	assert.Equal(t, mc.Duration, cfg.Monitoring.System.CPU.Maintenance.Duration, "CPU maintenance duration should be saved")
}

// TestService_Save_SystemMemoryMaintenance verifies that maintenance config is applied to the system memory config after Save.
func TestService_Save_SystemMemoryMaintenance(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := monitors.NewService(ctx, time.Second, time.Second, nil)

	mockMem := system.NewMockMemProvider(50)
	memMon := system.NewMem(90, "speedometer", 1, mockMem)
	mc := testMaintenance()
	svc.AddWithMaintenance(ctx, memMon, mc)

	cfg := &config.Config{}
	err := svc.Save(cfg)
	require.NoError(t, err)

	assert.Equal(t, mc.Cron, cfg.Monitoring.System.Memory.Maintenance.Cron, "memory maintenance cron should be saved")
	assert.Equal(t, mc.Duration, cfg.Monitoring.System.Memory.Maintenance.Duration, "memory maintenance duration should be saved")
}
