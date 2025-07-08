package config

import (
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	//go:embed test/bad.yaml
	badYaml string
)

func TestDefaultConfig(t *testing.T) {
	loader := NewLoader("test*.yaml", []string{os.TempDir()})
	cfg, err := loader.Load()
	require.NoError(t, err, "no error loading config")
	require.NotNil(t, cfg, "no config loaded")

	// web settings
	assert.Equal(t, "0.0.0.0", cfg.Web.Host, "cfg.web.host")
	assert.Equal(t, 8080, cfg.Web.Port, "cfg.web.port")
	assert.Equal(t, "LMON Dashboard", cfg.Monitoring.System.Title, "cfg.web.dashboardTitle")

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
	cfg.Monitoring.System.Title = "Test Dashboard"
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

func TestBadYaml(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	// set pre-edit config
	err := os.WriteFile(testFile, []byte(badYaml), 0644)
	assert.NoError(t, err, "Error creating initial file")

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	var cfgError viper.ConfigParseError
	assert.ErrorAs(t, err, &cfgError, "error loading config")
	assert.Nil(t, cfg, "no config loaded")

}

func TestNewLoader(t *testing.T) {
	type test struct {
		name        string
		cfgFilename string
		paths       []string
		envName     string
		envPaths    string
		expectName  string
		exepctPaths []string
		expectType  string
	}
	tests := []test{
		{
			name:        "config.yaml",
			cfgFilename: "config.yaml",
			paths:       nil,
			expectName:  "config",
			expectType:  "yaml",
			exepctPaths: []string{"."},
		},
		{
			name:        "config",
			cfgFilename: "config",
			paths:       nil,
			expectName:  "config",
			expectType:  "yaml",
			exepctPaths: []string{"."},
		},
		{
			name:        "empty",
			cfgFilename: "",
			paths:       nil,
			expectName:  "config",
			expectType:  "yaml",
			exepctPaths: []string{"."},
		},
		{
			name:        "/etc/lmon",
			cfgFilename: "",
			paths:       []string{"/etc/lmon"},
			expectName:  "config",
			expectType:  "yaml",
			exepctPaths: []string{"/etc/lmon"},
		},
		{
			name:        "/etc/lmon & ./config.env",
			cfgFilename: "./config.env",
			paths:       []string{"/etc/lmon"},
			expectName:  "config",
			expectType:  "env",
			exepctPaths: []string{"."},
		},
		{
			name:        "/var/lib/lmon/config.env",
			cfgFilename: "/var/lib/lmon/config.env",
			paths:       []string{"."},
			expectName:  "config",
			expectType:  "env",
			exepctPaths: []string{"/var/lib/lmon"},
		},
		{
			name:        "env config.yaml",
			envName:     "config.yaml",
			paths:       nil,
			expectName:  "config",
			expectType:  "yaml",
			exepctPaths: []string{"."},
		},
		{
			name:        "env ~/.lmon/config.yaml",
			envName:     "~/.lmon/config.yaml",
			paths:       nil,
			expectName:  "config",
			expectType:  "yaml",
			exepctPaths: []string{"~/.lmon"},
		},
		{
			name:        "env ~/.lmon & config.yaml",
			envName:     "config.yaml",
			envPaths:    "~/.lmon",
			paths:       nil,
			expectName:  "config",
			expectType:  "yaml",
			exepctPaths: []string{"~/.lmon"},
		},
		{
			name:        "env ~/.lmon & ./config.toml",
			envName:     "./config.toml",
			envPaths:    "'~/.lmon",
			paths:       nil,
			expectName:  "config",
			expectType:  "toml",
			exepctPaths: []string{"."},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envName != "" {
				t.Setenv("LMON_CONFIG_FILE", tt.envName)
			}
			if tt.envPaths != "" {
				t.Setenv("LMON_CONFIG_PATH", tt.envPaths)
			}
			loadder := NewLoader(tt.cfgFilename, tt.paths)
			assert.Equal(t, tt.expectName, loadder.name, "name")
			assert.Equal(t, tt.exepctPaths, loadder.paths, "paths")
			assert.Equal(t, tt.expectType, loadder.cfgType, "cfgType")
		})
	}
}

func Test_SanitiseName(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		want1 bool
	}{
		{"clean", "clean", false},
		{"clean-name", "clean-name", false},
		{"clean_name", "clean_name", false},
		{"clean name (sp)", "clean name (sp)", false},
		{"clean.name", "clean_name", true},
		{"    ", "unknown", true},
		{" ", "unknown", true},
		{" \t", "unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := SanitiseName(tt.name)
			assert.Equalf(t, tt.want, got, "SanitiseName(%v)", tt.name)
			assert.Equalf(t, tt.want1, got1, "SanitiseName(%v)", tt.name)
		})
	}
}
