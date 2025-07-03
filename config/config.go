package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Web       WebConfig       `mapstructure:"web"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	Webhook   WebhookConfig   `mapstructure:"webhook"`
}

// WebConfig represents the web server configuration
type WebConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// MonitoringConfig represents the monitoring configuration
type MonitoringConfig struct {
	Interval   int                `mapstructure:"interval"`
	Disk       []DiskConfig       `mapstructure:"disk"`
	System     SystemConfig       `mapstructure:"system"`
	Healthchecks []HealthcheckConfig `mapstructure:"healthchecks"`
}

// DiskConfig represents disk monitoring configuration
type DiskConfig struct {
	Path      string `mapstructure:"path"`
	Threshold int    `mapstructure:"threshold"`
	Icon      string `mapstructure:"icon"`
}

// SystemConfig represents system monitoring configuration
type SystemConfig struct {
	CPUThreshold    int    `mapstructure:"cputhreshold"`
	MemoryThreshold int    `mapstructure:"memorythreshold"`
	CPUIcon         string `mapstructure:"cpuicon"`
	MemoryIcon      string `mapstructure:"memoryicon"`
}

// HealthcheckConfig represents health check monitoring configuration
type HealthcheckConfig struct {
	Name      string `mapstructure:"name"`
	URL       string `mapstructure:"url"`
	Interval  int    `mapstructure:"interval"`
	Timeout   int    `mapstructure:"timeout"`
	Icon      string `mapstructure:"icon"`
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Web: WebConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Monitoring: MonitoringConfig{
			Interval: 60, // seconds
			Disk: []DiskConfig{
				{
					Path:      "/",
					Threshold: 80, // percentage
					Icon:      "storage",
				},
			},
			System: SystemConfig{
				CPUThreshold:    80, // percentage
				MemoryThreshold: 80, // percentage
				CPUIcon:         "memory",
				MemoryIcon:      "memory",
			},
			Healthchecks: []HealthcheckConfig{},
		},
		Webhook: WebhookConfig{
			Enabled: false,
			URL:     "",
		},
	}
}

// Load loads the configuration from file and environment variables
func Load() (*Config, error) {
	return LoadFromPaths([]string{".", "/etc/lmon"})
}

// LoadFromPaths loads the configuration from the specified paths
func LoadFromPaths(paths []string) (*Config, error) {
	config := DefaultConfig()

	// Set up Viper
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Add config paths
	for _, path := range paths {
		v.AddConfigPath(path)
	}

	// Set environment variable prefix
	v.SetEnvPrefix("LMON")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind environment variables
	bindEnvVars(v)

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal config
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return config, nil
}

// LoadFromFile loads the configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	config := DefaultConfig()

	// Set up Viper
	v := viper.New()
	v.SetConfigType("yaml")

	// Set environment variable prefix
	v.SetEnvPrefix("LMON")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind environment variables
	bindEnvVars(v)

	// Read the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}
	defer file.Close()

	// Read the config
	if err := v.ReadConfig(file); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal config
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return config, nil
}

// bindEnvVars binds environment variables to viper
func bindEnvVars(v *viper.Viper) {
	// Web config
	v.BindEnv("web.host", "LMON_WEB_HOST")
	v.BindEnv("web.port", "LMON_WEB_PORT")

	// Monitoring config
	v.BindEnv("monitoring.interval", "LMON_MONITORING_INTERVAL")

	// Webhook config
	v.BindEnv("webhook.enabled", "LMON_WEBHOOK_ENABLED")
	v.BindEnv("webhook.url", "LMON_WEBHOOK_URL")
}

// Save saves the configuration to a file
func Save(config *Config, path string) error {
	v := viper.New()
	v.SetConfigType("yaml")

	// Set the config values
	err := v.MergeConfigMap(map[string]interface{}{
		"web":        config.Web,
		"monitoring": config.Monitoring,
		"webhook":    config.Webhook,
	})
	if err != nil {
		return fmt.Errorf("failed to merge config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write config to file
	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
