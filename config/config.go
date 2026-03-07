package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Mode represents the operating mode of the lmon instance.
type Mode string

const (
	ModeNode       Mode = "node"
	ModeAggregator Mode = "aggregator"
)

// ResolveMode reads the LMON_MODE environment variable and returns the operating mode.
// Defaults to ModeNode if not set or unrecognized.
func ResolveMode() Mode {
	m := os.Getenv("LMON_MODE")
	switch Mode(m) {
	case ModeAggregator:
		return ModeAggregator
	default:
		return ModeNode
	}
}

// Config represents the application configuration
type Config struct {
	Web        WebConfig
	Monitoring MonitoringConfig
	Webhook    WebhookConfig
	Kubernetes KubernetesConfig
	Aggregator AggregatorConfig
	Database   DatabaseConfig
}

func SanitiseName(name string) (string, bool) {
	if strings.Contains(name, ".") {
		return strings.ReplaceAll(name, ".", "_"), true
	}
	if strings.TrimSpace(name) == "" {
		return "unknown", true
	}
	return name, false
}

// WebConfig represents the web server configuration
type WebConfig struct {
	Host string
	Port int
}

// MonitoringConfig represents the monitoring configuration
type MonitoringConfig struct {
	Interval                 int
	Disk                     map[string]DiskConfig
	System                   SystemConfig
	Healthcheck              map[string]HealthcheckConfig
	Ping                     map[string]PingConfig
	Docker                   map[string]DockerConfig
	K8sEvents                map[string]K8sEventsConfig  `mapstructure:"k8sevents"`
	K8sNodes                 map[string]K8sNodesConfig   `mapstructure:"k8snodes"`
	K8sService               map[string]K8sServiceConfig `mapstructure:"k8sservice"`
	AllowedRestartContainers []string                    // Global whitelist of container names/IDs allowed for restart operations
}

// DiskConfig represents disk monitoring configuration.
type DiskConfig struct {
	Threshold      int
	Icon           string
	Path           string
	AlertThreshold int // Number of consecutive failures before triggering alert (default: 1)
}

// SystemItem represents system monitoring configuration.
type SystemItem struct {
	Threshold      int
	Icon           string
	AlertThreshold int // Number of consecutive failures before triggering alert (default: 1)
}

// SystemConfig represents system monitoring configuration.
type SystemConfig struct {
	CPU    SystemItem
	Memory SystemItem
	Title  string
}

// HealthcheckConfig represents health check monitoring configuration
type HealthcheckConfig struct {
	URL               string
	Timeout           int
	RespCode          int
	Icon              string
	RestartContainers string `json:"restart_containers,omitempty"` // Optional: comma-separated list of containers to restart
	AlertThreshold    int    // Number of consecutive failures before triggering alert (default: 1)
}

// PingConfig represents ping monitor configuration
type PingConfig struct {
	Address        string
	Timeout        int
	Icon           string
	AmberThreshold int // Response time in ms for amber status (required)
	AlertThreshold int // Number of consecutive failures before triggering alert (default: 1)
}

// DockerConfig represents Docker container monitoring configuration
type DockerConfig struct {
	Containers     string // Space or comma-separated list of container names/IDs
	Threshold      int    // Max restart count threshold before alerting
	Icon           string
	AlertThreshold int // Number of consecutive failures before triggering alert (default: 1)
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	Enabled bool
	URL     string
}

// KubernetesConfig represents Kubernetes integration configuration
type KubernetesConfig struct {
	Enabled    bool
	InCluster  bool   `mapstructure:"in_cluster"`
	Kubeconfig string
	Namespace  string
}

// AggregatorConfig represents aggregator mode configuration
type AggregatorConfig struct {
	NodeLabel       string `mapstructure:"node_label"`
	NodePort        int    `mapstructure:"node_port"`
	NodeMetricsPath string `mapstructure:"node_metrics_path"`
	ScrapeInterval  int    `mapstructure:"scrape_interval"`
}

// DatabaseConfig represents optional PostgreSQL metrics recording configuration
type DatabaseConfig struct {
	URL           string
	RetentionDays int `mapstructure:"retention_days"`
	BatchSize     int `mapstructure:"batch_size"`
	WriteInterval int `mapstructure:"write_interval"` // seconds between DB writes, 0 = every check
	PruneInterval int `mapstructure:"prune_interval"` // minutes between prune runs, default 60
}

// K8sEventsConfig represents Kubernetes events monitoring configuration
type K8sEventsConfig struct {
	Namespaces     string
	Threshold      int
	Window         int
	Icon           string
	AlertThreshold int
}

// K8sNodesConfig represents Kubernetes nodes monitoring configuration
type K8sNodesConfig struct {
	Icon           string
	AlertThreshold int
}

// K8sServiceConfig represents Kubernetes service pod health monitoring configuration
type K8sServiceConfig struct {
	Namespace      string
	Service        string
	HealthPath     string `mapstructure:"health_path"`
	Port           int
	Threshold      int // % healthy pods for green
	Timeout        int
	Icon           string
	AlertThreshold int
}

type Loader struct {
	v       *viper.Viper
	paths   []string
	name    string
	cfgType string
}

// NewLoader creates a new Loader instance.
// If cfgFilename is empty, it defaults to "config.yaml".
// If cfgFilename is not empty, it will be used as the config file name.
// If paths is nil or empty, it defaults to the current directory.
// Passed in values can be overridden by environment variables LMON_CONFIG_PATH and LMON_CONFIG_FILE.
// If LMON_CONFIG_FILE contains a path and config file, the path will be used as the config file path.
func NewLoader(cfgFilename string, paths []string) *Loader {
	envPath := os.Getenv("LMON_CONFIG_PATH")
	envName := os.Getenv("LMON_CONFIG_FILE")

	// If envName is not empty, use it as the config file cfgFilename
	if envName != "" {
		cfgFilename = envName
	}

	// Default config file cfgFilename
	if cfgFilename == "" {
		cfgFilename = "config.yaml"
	}

	// If envPath is not empty, use it as our only path.
	if envPath != "" {
		paths = []string{envPath}
	}

	// does envName have a path and config file? If so, extract them.
	if strings.Contains(cfgFilename, string(os.PathSeparator)) {
		p := filepath.Dir(cfgFilename)
		cfgFilename = filepath.Base(cfgFilename)
		if len(p) > 0 {
			paths = []string{p}
		}
	}

	// If paths is nil or empty, set it to the current directory
	if len(paths) == 0 {
		paths = []string{"."}
	}

	parts := strings.Split(cfgFilename, ".")
	cfgType := "yaml"
	if len(parts) > 1 {
		cfgType = parts[len(parts)-1]
		cfgFilename = strings.Join(parts[:len(parts)-1], ".")
	}

	return &Loader{
		v:       viper.New(),
		paths:   paths,
		name:    cfgFilename,
		cfgType: cfgType,
	}
}

func (l *Loader) init() {
	// Set up Viper

	l.v.SetConfigType(l.cfgType)
	l.v.SetConfigName(l.name)

	// Add config paths
	for _, path := range l.paths {
		l.v.AddConfigPath(path)
	}

	// Set environment variable prefix
	l.v.SetEnvPrefix("LMON")
	l.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	l.v.AutomaticEnv()

	// Set defualts
	l.setDefaults()
}

// Load loads the configuration from file and environment variables
func (l *Loader) Load() (*Config, error) {
	config := Config{}

	l.init()

	// Try to read config file
	if err := l.v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use defaults and env vars
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal config
	if err := l.v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	if config.Monitoring.Disk == nil {
		config.Monitoring.Disk = make(map[string]DiskConfig)
	}
	if config.Monitoring.Healthcheck == nil {
		config.Monitoring.Healthcheck = make(map[string]HealthcheckConfig)
	}
	if config.Monitoring.Ping == nil {
		config.Monitoring.Ping = make(map[string]PingConfig)
	}
	if config.Monitoring.Docker == nil {
		config.Monitoring.Docker = make(map[string]DockerConfig)
	}
	if config.Monitoring.K8sEvents == nil {
		config.Monitoring.K8sEvents = make(map[string]K8sEventsConfig)
	}
	if config.Monitoring.K8sNodes == nil {
		config.Monitoring.K8sNodes = make(map[string]K8sNodesConfig)
	}
	if config.Monitoring.K8sService == nil {
		config.Monitoring.K8sService = make(map[string]K8sServiceConfig)
	}

	// Apply defaults for new config sections (not via viper to avoid polluting saved files)
	applyKubernetesDefaults(&config.Kubernetes)
	applyAggregatorDefaults(&config.Aggregator)
	applyDatabaseDefaults(&config.Database)

	return &config, nil
}

func applyKubernetesDefaults(cfg *KubernetesConfig) {
	// InCluster defaults to true if kubernetes is enabled and no kubeconfig is set
	if cfg.Enabled && cfg.Kubeconfig == "" {
		cfg.InCluster = true
	}
	// If nothing is set at all, default InCluster to true for when it's eventually enabled
	if !cfg.Enabled && cfg.Kubeconfig == "" && !cfg.InCluster {
		cfg.InCluster = true
	}
}

func applyAggregatorDefaults(cfg *AggregatorConfig) {
	if cfg.NodePort == 0 {
		cfg.NodePort = 8080
	}
	if cfg.NodeMetricsPath == "" {
		cfg.NodeMetricsPath = "/metrics"
	}
	if cfg.ScrapeInterval == 0 {
		cfg.ScrapeInterval = 30
	}
}

func applyDatabaseDefaults(cfg *DatabaseConfig) {
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 7
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1000
	}
	// WriteInterval 0 means "every check" - valid default, no action needed
	if cfg.PruneInterval == 0 {
		cfg.PruneInterval = 60
	}
}

func (l *Loader) setDefaults() {
	l.v.SetDefault("web.host", "0.0.0.0")
	l.v.SetDefault("web.port", 8080)
	l.v.SetDefault("monitoring.interval", 60)

	l.v.SetDefault("monitoring.system.cpu.threshold", 90)
	l.v.SetDefault("monitoring.system.memory.threshold", 90)
	l.v.SetDefault("monitoring.system.cpu.icon", "cpu")
	l.v.SetDefault("monitoring.system.memory.icon", "speedometer")
	l.v.SetDefault("monitoring.system.title", "LMON Dashboard")

	l.v.SetDefault("webhook.enabled", true)
	l.v.SetDefault("webhook.url", "http://localhost:8080/testhook")
}

// Save saves the configuration to a file
func (l *Loader) Save(config *Config) error {
	// create a new viper clearing the disk and healthcheck maps
	l.v = viper.New()
	l.init()

	l.v.Set("web.host", config.Web.Host)
	l.v.Set("web.port", config.Web.Port)
	l.v.Set("monitoring.interval", config.Monitoring.Interval)

	l.v.Set("monitoring.system.cpu.threshold", config.Monitoring.System.CPU.Threshold)
	l.v.Set("monitoring.system.memory.threshold", config.Monitoring.System.Memory.Threshold)
	l.v.Set("monitoring.system.cpu.icon", config.Monitoring.System.CPU.Icon)
	l.v.Set("monitoring.system.memory.icon", config.Monitoring.System.Memory.Icon)
	l.v.Set("monitoring.system.title", config.Monitoring.System.Title)
	// Always save alertthreshold, defaulting to 1 if not set
	cpuAlert := config.Monitoring.System.CPU.AlertThreshold
	if cpuAlert <= 0 {
		cpuAlert = 1
	}
	l.v.Set("monitoring.system.cpu.alertthreshold", cpuAlert)
	memAlert := config.Monitoring.System.Memory.AlertThreshold
	if memAlert <= 0 {
		memAlert = 1
	}
	l.v.Set("monitoring.system.memory.alertthreshold", memAlert)

	// Save the global allowed restart containers list
	if len(config.Monitoring.AllowedRestartContainers) > 0 {
		l.v.Set("monitoring.allowedrestartcontainers", config.Monitoring.AllowedRestartContainers)
	}

	l.v.Set("webhook.enabled", config.Webhook.Enabled)
	l.v.Set("webhook.url", config.Webhook.URL)

	for name, disk := range config.Monitoring.Disk {
		l.v.Set(fmt.Sprintf("monitoring.disk.%s.path", name), disk.Path)
		l.v.Set(fmt.Sprintf("monitoring.disk.%s.threshold", name), disk.Threshold)
		l.v.Set(fmt.Sprintf("monitoring.disk.%s.icon", name), disk.Icon)
		// Always save alertthreshold, defaulting to 1 if not set
		alertThreshold := disk.AlertThreshold
		if alertThreshold <= 0 {
			alertThreshold = 1
		}
		l.v.Set(fmt.Sprintf("monitoring.disk.%s.alertthreshold", name), alertThreshold)
	}

	for name, healthcheck := range config.Monitoring.Healthcheck {
		l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.url", name), healthcheck.URL)
		l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.timeout", name), healthcheck.Timeout)
		l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.respcode", name), healthcheck.RespCode)
		l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.icon", name), healthcheck.Icon)
		if healthcheck.RestartContainers != "" {
			l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.restartcontainers", name), healthcheck.RestartContainers)
		}
		// Always save alertthreshold, defaulting to 1 if not set
		alertThreshold := healthcheck.AlertThreshold
		if alertThreshold <= 0 {
			alertThreshold = 1
		}
		l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.alertthreshold", name), alertThreshold)
	}

	for name, ping := range config.Monitoring.Ping {
		l.v.Set(fmt.Sprintf("monitoring.ping.%s.address", name), ping.Address)
		l.v.Set(fmt.Sprintf("monitoring.ping.%s.timeout", name), ping.Timeout)
		l.v.Set(fmt.Sprintf("monitoring.ping.%s.icon", name), ping.Icon)
		l.v.Set(fmt.Sprintf("monitoring.ping.%s.amberthreshold", name), ping.AmberThreshold)
		// Always save alertthreshold, defaulting to 1 if not set
		alertThreshold := ping.AlertThreshold
		if alertThreshold <= 0 {
			alertThreshold = 1
		}
		l.v.Set(fmt.Sprintf("monitoring.ping.%s.alertthreshold", name), alertThreshold)
	}

	for name, docker := range config.Monitoring.Docker {
		l.v.Set(fmt.Sprintf("monitoring.docker.%s.containers", name), docker.Containers)
		l.v.Set(fmt.Sprintf("monitoring.docker.%s.threshold", name), docker.Threshold)
		l.v.Set(fmt.Sprintf("monitoring.docker.%s.icon", name), docker.Icon)
		// Always save alertthreshold, defaulting to 1 if not set
		alertThreshold := docker.AlertThreshold
		if alertThreshold <= 0 {
			alertThreshold = 1
		}
		l.v.Set(fmt.Sprintf("monitoring.docker.%s.alertthreshold", name), alertThreshold)
	}

	// Save Kubernetes config if enabled
	if config.Kubernetes.Enabled {
		l.v.Set("kubernetes.enabled", config.Kubernetes.Enabled)
		l.v.Set("kubernetes.in_cluster", config.Kubernetes.InCluster)
		if config.Kubernetes.Kubeconfig != "" {
			l.v.Set("kubernetes.kubeconfig", config.Kubernetes.Kubeconfig)
		}
		if config.Kubernetes.Namespace != "" {
			l.v.Set("kubernetes.namespace", config.Kubernetes.Namespace)
		}
	}

	// Save Aggregator config if node_label is set
	if config.Aggregator.NodeLabel != "" {
		l.v.Set("aggregator.node_label", config.Aggregator.NodeLabel)
		l.v.Set("aggregator.node_port", config.Aggregator.NodePort)
		l.v.Set("aggregator.node_metrics_path", config.Aggregator.NodeMetricsPath)
		l.v.Set("aggregator.scrape_interval", config.Aggregator.ScrapeInterval)
	}

	// Save Database config if URL is set
	if config.Database.URL != "" {
		l.v.Set("database.url", config.Database.URL)
		l.v.Set("database.retention_days", config.Database.RetentionDays)
		l.v.Set("database.batch_size", config.Database.BatchSize)
		l.v.Set("database.write_interval", config.Database.WriteInterval)
		l.v.Set("database.prune_interval", config.Database.PruneInterval)
	}

	// Save K8s monitors
	for name, k8sEvents := range config.Monitoring.K8sEvents {
		l.v.Set(fmt.Sprintf("monitoring.k8sevents.%s.namespaces", name), k8sEvents.Namespaces)
		l.v.Set(fmt.Sprintf("monitoring.k8sevents.%s.threshold", name), k8sEvents.Threshold)
		l.v.Set(fmt.Sprintf("monitoring.k8sevents.%s.window", name), k8sEvents.Window)
		l.v.Set(fmt.Sprintf("monitoring.k8sevents.%s.icon", name), k8sEvents.Icon)
		alertThreshold := k8sEvents.AlertThreshold
		if alertThreshold <= 0 {
			alertThreshold = 1
		}
		l.v.Set(fmt.Sprintf("monitoring.k8sevents.%s.alertthreshold", name), alertThreshold)
	}

	for name, k8sNodes := range config.Monitoring.K8sNodes {
		l.v.Set(fmt.Sprintf("monitoring.k8snodes.%s.icon", name), k8sNodes.Icon)
		alertThreshold := k8sNodes.AlertThreshold
		if alertThreshold <= 0 {
			alertThreshold = 1
		}
		l.v.Set(fmt.Sprintf("monitoring.k8snodes.%s.alertthreshold", name), alertThreshold)
	}

	for name, k8sSvc := range config.Monitoring.K8sService {
		l.v.Set(fmt.Sprintf("monitoring.k8sservice.%s.namespace", name), k8sSvc.Namespace)
		l.v.Set(fmt.Sprintf("monitoring.k8sservice.%s.service", name), k8sSvc.Service)
		l.v.Set(fmt.Sprintf("monitoring.k8sservice.%s.health_path", name), k8sSvc.HealthPath)
		l.v.Set(fmt.Sprintf("monitoring.k8sservice.%s.port", name), k8sSvc.Port)
		l.v.Set(fmt.Sprintf("monitoring.k8sservice.%s.threshold", name), k8sSvc.Threshold)
		l.v.Set(fmt.Sprintf("monitoring.k8sservice.%s.timeout", name), k8sSvc.Timeout)
		l.v.Set(fmt.Sprintf("monitoring.k8sservice.%s.icon", name), k8sSvc.Icon)
		alertThreshold := k8sSvc.AlertThreshold
		if alertThreshold <= 0 {
			alertThreshold = 1
		}
		l.v.Set(fmt.Sprintf("monitoring.k8sservice.%s.alertthreshold", name), alertThreshold)
	}

	// overwrite the config file or create it if it doesn't exist
	err := l.v.WriteConfig()
	var configFileNotFoundError viper.ConfigFileNotFoundError
	ok := errors.As(err, &configFileNotFoundError)
	if err != nil && ok {
		err = l.v.SafeWriteConfig()
	}
	if err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}
	return nil
}

func (l *Loader) FilePath() string {
	return l.v.ConfigFileUsed()
}
