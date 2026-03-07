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

	//go:embed test/sample_ping.yaml
	samplePingYaml string

	//go:embed test/sample_docker.yaml
	sampleDockerYaml string

	//go:embed test/sample_k8s.yaml
	sampleK8sYaml string

	//go:embed test/sample_database.yaml
	sampleDatabaseYaml string

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
	assert.Equal(t, "http://localhost:8080/testhook", cfg.Webhook.URL, "cfg.webhook.url")

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

	cfg.Monitoring.Disk["test"] = DiskConfig{75, "test", "/test", 0}
	saveAndCheckContent(t, loader, cfg, testFile, changedYamlAddDisk)
}

func TestAddPing(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))
	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	cfg.Monitoring.Ping["ping_sample"] = PingConfig{
		Address:        "8.8.8.8",
		Timeout:        1000,
		Icon:           "wifi",
		AmberThreshold: 100,
	}
	saveAndCheckContent(t, loader, cfg, testFile, samplePingYaml)
}

func TestAddDocker(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))
	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	cfg.Monitoring.Docker["app_containers"] = DockerConfig{
		Containers: "web-app, api-server",
		Threshold:  5,
		Icon:       "box",
	}
	saveAndCheckContent(t, loader, cfg, testFile, sampleDockerYaml)
}

func TestAddHealthcheck(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))
	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	assert.NoError(t, err, "no error loading config")

	cfg.Monitoring.Healthcheck["test"] = HealthcheckConfig{"http://localhost:8080/test", 10, 401, "test", "", 0}
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

	cfg.Monitoring.Disk["test"] = DiskConfig{75, "test", "/test", 0}
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
		Threshold:      75,
		Icon:           "test",
		Path:           "/root",
		AlertThreshold: 0,
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

	cfg.Monitoring.Healthcheck["test"] = HealthcheckConfig{"http://localhost:8080/test", 10, 200, "test", "", 0}
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

	cfg.Monitoring.Healthcheck["self"] = HealthcheckConfig{"http://localhost:8080/test", 10, 200, "test", "", 0}

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

func TestK8sConfigLoad(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	err := os.WriteFile(testFile, []byte(sampleK8sYaml), 0644)
	assert.NoError(t, err, "Error creating initial file")

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	require.NoError(t, err, "no error loading config")
	require.NotNil(t, cfg, "config loaded")

	// Kubernetes config
	assert.True(t, cfg.Kubernetes.Enabled, "kubernetes.enabled")
	assert.False(t, cfg.Kubernetes.InCluster, "kubernetes.in_cluster")
	assert.Equal(t, "/home/user/.kube/config", cfg.Kubernetes.Kubeconfig, "kubernetes.kubeconfig")
	assert.Equal(t, "default", cfg.Kubernetes.Namespace, "kubernetes.namespace")

	// K8s Events
	assert.Equal(t, 1, len(cfg.Monitoring.K8sEvents), "k8sevents count")
	events, ok := cfg.Monitoring.K8sEvents["default-events"]
	assert.True(t, ok, "k8sevents entry exists")
	assert.Equal(t, "default", events.Namespaces)
	assert.Equal(t, 5, events.Threshold)
	assert.Equal(t, 300, events.Window)
	assert.Equal(t, "lightning", events.Icon)

	// K8s Nodes
	assert.Equal(t, 1, len(cfg.Monitoring.K8sNodes), "k8snodes count")
	nodes, ok := cfg.Monitoring.K8sNodes["cluster-nodes"]
	assert.True(t, ok, "k8snodes entry exists")
	assert.Equal(t, "hdd-rack", nodes.Icon)

	// K8s Service
	assert.Equal(t, 1, len(cfg.Monitoring.K8sService), "k8sservice count")
	svc, ok := cfg.Monitoring.K8sService["web-app"]
	assert.True(t, ok, "k8sservice entry exists")
	assert.Equal(t, "production", svc.Namespace)
	assert.Equal(t, "web-frontend", svc.Service)
	assert.Equal(t, "/healthz", svc.HealthPath)
	assert.Equal(t, 8080, svc.Port)
	assert.Equal(t, 80, svc.Threshold)
	assert.Equal(t, 5, svc.Timeout)
	assert.Equal(t, "globe", svc.Icon)
}

func TestK8sConfigSave(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	require.NoError(t, err, "no error loading config")

	cfg.Kubernetes = KubernetesConfig{
		Enabled:    true,
		InCluster:  false,
		Kubeconfig: "/home/user/.kube/config",
		Namespace:  "default",
	}

	cfg.Monitoring.K8sEvents = map[string]K8sEventsConfig{
		"default-events": {
			Namespaces: "default",
			Threshold:  5,
			Window:     300,
			Icon:       "lightning",
		},
	}
	cfg.Monitoring.K8sNodes = map[string]K8sNodesConfig{
		"cluster-nodes": {
			Icon: "hdd-rack",
		},
	}
	cfg.Monitoring.K8sService = map[string]K8sServiceConfig{
		"web-app": {
			Namespace:  "production",
			Service:    "web-frontend",
			HealthPath: "/healthz",
			Port:       8080,
			Threshold:  80,
			Timeout:    5,
			Icon:       "globe",
		},
	}

	saveAndCheckContent(t, loader, cfg, testFile, sampleK8sYaml)
}

func TestDatabaseConfigLoad(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	err := os.WriteFile(testFile, []byte(sampleDatabaseYaml), 0644)
	assert.NoError(t, err, "Error creating initial file")

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	require.NoError(t, err, "no error loading config")
	require.NotNil(t, cfg, "config loaded")

	assert.Equal(t, "postgres://user:pass@localhost:5432/lmon", cfg.Database.URL)
	assert.Equal(t, 14, cfg.Database.RetentionDays)
	assert.Equal(t, 500, cfg.Database.BatchSize)
	assert.Equal(t, 10, cfg.Database.WriteInterval)
	assert.Equal(t, 30, cfg.Database.PruneInterval)
}

func TestDatabaseConfigSave(t *testing.T) {
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	require.NoError(t, err, "no error loading config")

	cfg.Database = DatabaseConfig{
		URL:           "postgres://user:pass@localhost:5432/lmon",
		RetentionDays: 14,
		BatchSize:     500,
		WriteInterval: 10,
		PruneInterval: 30,
	}

	saveAndCheckContent(t, loader, cfg, testFile, sampleDatabaseYaml)
}

func TestModeResolution(t *testing.T) {
	t.Run("default is node", func(t *testing.T) {
		t.Setenv("LMON_MODE", "")
		assert.Equal(t, ModeNode, ResolveMode())
	})

	t.Run("explicit node", func(t *testing.T) {
		t.Setenv("LMON_MODE", "node")
		assert.Equal(t, ModeNode, ResolveMode())
	})

	t.Run("aggregator mode", func(t *testing.T) {
		t.Setenv("LMON_MODE", "aggregator")
		assert.Equal(t, ModeAggregator, ResolveMode())
	})

	t.Run("unknown defaults to node", func(t *testing.T) {
		t.Setenv("LMON_MODE", "unknown")
		assert.Equal(t, ModeNode, ResolveMode())
	})
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that loading a config without the new fields works fine
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	// Write an old-style config without k8s/database sections
	err := os.WriteFile(testFile, []byte(defaultYaml), 0644)
	assert.NoError(t, err)

	loader := NewLoader("test.yaml", []string{dir})
	cfg, err := loader.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// New fields should have zero/default values
	assert.False(t, cfg.Kubernetes.Enabled)
	assert.True(t, cfg.Kubernetes.InCluster) // default is true
	assert.Equal(t, "", cfg.Database.URL)
	assert.Equal(t, 7, cfg.Database.RetentionDays) // default
	assert.Equal(t, 1000, cfg.Database.BatchSize)   // default
	assert.Equal(t, 8080, cfg.Aggregator.NodePort)   // default
	assert.Equal(t, "/metrics", cfg.Aggregator.NodeMetricsPath) // default
	assert.Equal(t, 30, cfg.Aggregator.ScrapeInterval) // default

	// Maps should be initialized but empty
	assert.NotNil(t, cfg.Monitoring.K8sEvents)
	assert.Equal(t, 0, len(cfg.Monitoring.K8sEvents))
	assert.NotNil(t, cfg.Monitoring.K8sNodes)
	assert.Equal(t, 0, len(cfg.Monitoring.K8sNodes))
	assert.NotNil(t, cfg.Monitoring.K8sService)
	assert.Equal(t, 0, len(cfg.Monitoring.K8sService))

	// Saving should produce the same output as the original
	saveAndCheckContent(t, loader, cfg, testFile, defaultYaml)
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
