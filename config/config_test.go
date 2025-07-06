package config

import (
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	loader := NewLoader("test*.yaml", []string{os.TempDir()})
	cfg, err := loader.Load()
	require.NoError(t, err, "no error loading config")
	require.NotNil(t, cfg, "no config loaded")

	// web settings
	assert.Equal(t, "0.0.0.0", cfg.Web.Host, "cfg.web.host")
	assert.Equal(t, 8080, cfg.Web.Port, "cfg.web.port")
	assert.Equal(t, "LMON Dashboard", cfg.Web.Title, "cfg.web.dashboardTitle")

	// monitoring settings
	assert.Equal(t, 60, cfg.Monitoring.Interval, "cfg.monitoring.interval")

	// System settings
	assert.Equal(t, 90, cfg.Monitoring.System.CPU.Threshold, "cfg.monitoring.system.cpu.threshold")
	assert.Equal(t, 90, cfg.Monitoring.System.Memory.Threshold, "cfg.monitoring.system.memory.threshold")
	assert.Equal(t, "cpu", cfg.Monitoring.System.CPU.Icon, "cfg.monitoring.system.cpu.icon")
	assert.Equal(t, "speedometer", cfg.Monitoring.System.Memory.Icon, "cfg.monitoring.system.memory.icon")

	// Webhook settings
	assert.True(t, cfg.Webhook.Enabled, "cfg.webhook.enabled")
	assert.Equal(t, "http://localhost:8080/test_webhook", cfg.Webhook.URL, "cfg.webhook.url")

	// Disk settings
	assert.Equal(t, 0, len(cfg.Monitoring.Disk), "len(disk)")
	// Healthcheck settings
	assert.Equal(t, 0, len(cfg.Monitoring.Healthcheck), "len(healthcheck)")
}

var (
	//go:embed test/default.yaml
	defaultYaml string

	//go:embed test/changed.yaml
	changedYaml string

	//go:embed test/changed_add_disk.yaml
	changedYamlAddDisk string

	//go:embed test/changed_edit_before.yaml
	changedYamlEditBefore string

	//go:embed test/changed_edit_disk.yaml
	changedYamlEditDisk string

	//go:embed test/changed_add_remove_disk.yaml
	changedYamlAddRemoveDisk string

	//go:embed test/changed_add_healthcheck.yaml
	changedYamlAddHealthcheck string
	//go:embed test/changed_add_remove_healthcheck.yaml
	changedYamlAddRemoveHealthcheck string
	//go:embed test/changed_edit_healthcheck.yaml
	changedYamlEditHealthcheck string

	//go:embed test/default.env
	defaultEnv string
)

func TestSaveDefaults(t *testing.T) {
	testFiles := []string{"test.yaml", "test.env"}
	expect := []string{defaultYaml, defaultEnv}
	for i, f := range testFiles {
		t.Run(f, func(t *testing.T) {
			dir := t.TempDir()
			testFile := strings.Join([]string{dir, f}, string(os.PathSeparator))

			loader := NewLoader(testFile, nil)
			cfg, err := loader.Load()
			assert.NoError(t, err, "no error loading config")
			require.NotNil(t, cfg, "no config loaded")

			saveAndCheckContent(t, loader, cfg, testFile, expect[i])
		})
	}
}

func TestChangeValues(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))
	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	cfg.Web.Host = "localhost"
	cfg.Web.Port = 8000
	cfg.Web.Title = "Test Dashboard"
	cfg.Webhook.Enabled = false
	cfg.Webhook.URL = "bad url"
	cfg.Monitoring.System.CPU.Threshold = 99
	cfg.Monitoring.System.CPU.Icon = "cpuicon"
	cfg.Monitoring.System.Memory.Threshold = 98
	cfg.Monitoring.System.Memory.Icon = "memicon"
	cfg.Monitoring.Interval = 15

	saveAndCheckContent(t, loader, cfg, testFile, changedYaml)
}

func saveAndCheckContent(t *testing.T, loader *Loader, cfg *Config, testFile string, content string) {
	t.Helper()
	err := loader.Save(cfg)
	assert.NoError(t, err, "saving config")
	assert.FileExistsf(t, testFile, "output file exists")
	body, err := os.ReadFile(testFile)
	require.NoError(t, err, "no error reading output file")
	bodyStr := string(body)
	assert.Equal(t, content, bodyStr, "output file contents")
}

func TestAddFolders(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))
	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	cfg.Monitoring.Disk["test"] = DiskConfig{75, "test", "/test"}
	saveAndCheckContent(t, loader, cfg, testFile, changedYamlAddDisk)
}

func TestAddHealthcheck(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))
	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	cfg.Monitoring.Healthcheck["test"] = HealthcheckConfig{"http://localhost:8080/test", 10, "test"}
	saveAndCheckContent(t, loader, cfg, testFile, changedYamlAddHealthcheck)
}

func TestRemoveFolder(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	// set pre-edit config
	err := os.WriteFile(testFile, []byte(changedYamlEditBefore), 0644)
	assert.NoError(t, err, "Error creating initial file")

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	// we already have the one to remove?
	assert.Equal(t, 1, len(cfg.Monitoring.Disk))

	cfg.Monitoring.Disk["test"] = DiskConfig{75, "test", "/test"}
	delete(cfg.Monitoring.Disk, "root")

	saveAndCheckContent(t, loader, cfg, testFile, changedYamlAddRemoveDisk)
}

func TestEditFolder(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	// set pre-edit config
	err := os.WriteFile(testFile, []byte(changedYamlEditBefore), 0644)
	assert.NoError(t, err, "Error creating initial file")

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	// we already have the one to edit?
	assert.Equal(t, 1, len(cfg.Monitoring.Disk))

	cfg.Monitoring.Disk["root"] = DiskConfig{
		Threshold: 75,
		Icon:      "test",
		Path:      "/root",
	}

	saveAndCheckContent(t, loader, cfg, testFile, changedYamlEditDisk)
}

func TestRemoveHealthCheck(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	// set pre-edit config
	err := os.WriteFile(testFile, []byte(changedYamlEditBefore), 0644)
	assert.NoError(t, err, "Error creating initial file")

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	// we already have the one to edit?
	assert.Equal(t, 1, len(cfg.Monitoring.Healthcheck))

	cfg.Monitoring.Healthcheck["test"] = HealthcheckConfig{"http://localhost:8080/test", 10, "test"}
	delete(cfg.Monitoring.Healthcheck, "self")

	saveAndCheckContent(t, loader, cfg, testFile, changedYamlAddRemoveHealthcheck)
}

func TestLoadChangeAndSave(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	// set pre-edit config
	err := os.WriteFile(testFile, []byte(changedYamlEditBefore), 0644)
	assert.NoError(t, err, "Error creating initial file")

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	// we already have the one to edit?
	assert.Equal(t, 1, len(cfg.Monitoring.Healthcheck))

	cfg.Monitoring.Healthcheck["self"] = HealthcheckConfig{"http://localhost:8080/test", 10, "test"}

	saveAndCheckContent(t, loader, cfg, testFile, changedYamlEditHealthcheck)

}
