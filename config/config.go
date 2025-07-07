package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Web        WebConfig
	Monitoring MonitoringConfig
	Webhook    WebhookConfig
}

// WebConfig represents the web server configuration
type WebConfig struct {
	Host string
	Port int
}

// MonitoringConfig represents the monitoring configuration
type MonitoringConfig struct {
	Interval    int
	Disk        map[string]DiskConfig
	System      SystemConfig
	Healthcheck map[string]HealthcheckConfig
}

// DiskConfig represents disk monitoring configuration.
type DiskConfig struct {
	Threshold int
	Icon      string
	Path      string
}

// SystemItem represents system monitoring configuration.
type SystemItem struct {
	Threshold int
	Icon      string
}

// SystemConfig represents system monitoring configuration.
type SystemConfig struct {
	CPU    SystemItem
	Memory SystemItem
	Title  string
}

// HealthcheckConfig represents health check monitoring configuration
type HealthcheckConfig struct {
	URL     string
	Timeout int
	Icon    string
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	Enabled bool
	URL     string
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
	if paths == nil || len(paths) == 0 {
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

	return &config, nil
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
	l.v.SetDefault("webhook.url", "http://localhost:8080/test_webhook")

	// l.v.SetDefault("monitoring.disk.root.path", "/")
	// l.v.SetDefault("monitoring.disk.root.threshold", 80)
	// l.v.SetDefault("monitoring.disk.root.icon", "storage")
	//
	// l.v.SetDefault("monitoring.healthcheck.self.url", "http://localhost:8080/healthz")
	// l.v.SetDefault("monitoring.healthcheck.self.timeout", 5)
	// l.v.SetDefault("monitoring.healthcheck.self.icon", "activity")
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

	l.v.Set("webhook.enabled", config.Webhook.Enabled)
	l.v.Set("webhook.url", config.Webhook.URL)

	for name, disk := range config.Monitoring.Disk {
		l.v.Set(fmt.Sprintf("monitoring.disk.%s.path", name), disk.Path)
		l.v.Set(fmt.Sprintf("monitoring.disk.%s.threshold", name), disk.Threshold)
		l.v.Set(fmt.Sprintf("monitoring.disk.%s.icon", name), disk.Icon)
	}

	for name, healthcheck := range config.Monitoring.Healthcheck {
		l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.url", name), healthcheck.URL)
		l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.timeout", name), healthcheck.Timeout)
		l.v.Set(fmt.Sprintf("monitoring.healthcheck.%s.icon", name), healthcheck.Icon)
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

func (l *Loader) FilePath() interface{} {
	return l.v.ConfigFileUsed()
}
