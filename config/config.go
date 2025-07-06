package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Web        WebConfig        `mapstructure:"web" json:"web"`
	Monitoring MonitoringConfig `mapstructure:"monitoring" json:"monitoring"`
	Webhook    WebhookConfig    `mapstructure:"webhook" json:"webhook"`
}

// WebConfig represents the web server configuration
type WebConfig struct {
	Host           string `mapstructure:"host" json:"host"`
	Port           int    `mapstructure:"port" json:"port"`
	DashboardTitle string `mapstructure:"dashboard_title" json:"dashboard_title"`
}

// MonitoringConfig represents the monitoring configuration
type MonitoringConfig struct {
	Interval     int                 `mapstructure:"interval" json:"interval"`
	Disk         []DiskConfig        `mapstructure:"disk" json:"disk"`
	System       SystemConfig        `mapstructure:"system" json:"system"`
	Healthchecks []HealthcheckConfig `mapstructure:"healthchecks" json:"healthchecks"`
}

type CPUItem struct {
	Threshold int    `mapstructure:"threshold" json:"threshold"`
	Icon      string `mapstructure:"icon" json:"icon"`
}

// DiskConfig represents disk monitoring configuration
type DiskConfig struct {
	Threshold int    `mapstructure:"threshold" json:"threshold"`
	Icon      string `mapstructure:"icon" json:"icon"`
	Path      string `mapstructure:"path" json:"path"`
}

// SystemConfig represents system monitoring configuration
type SystemConfig struct {
	CPU    CPUItem `mapstructure:"cpu" json:"cpu"`
	Memory CPUItem `mapstructure:"memory" json:"memory"`
}

// HealthcheckConfig represents health check monitoring configuration
type HealthcheckConfig struct {
	Name     string `mapstructure:"name" json:"name"`
	URL      string `mapstructure:"url" json:"url"`
	Interval int    `mapstructure:"interval" json:"interval"`
	Timeout  int    `mapstructure:"timeout" json:"timeout"`
	Icon     string `mapstructure:"icon" json:"icon"`
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	URL     string `mapstructure:"url" json:"url"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Web: WebConfig{
			Host:           "0.0.0.0",
			Port:           8080,
			DashboardTitle: "Monitoring Dashboard",
		},
		Monitoring: MonitoringConfig{
			Interval: 60, // seconds
			Disk: []DiskConfig{
				{
					Threshold: 80, // percentage
					Icon:      "storage",
					Path:      "/",
				},
			},
			System: SystemConfig{
				CPU: CPUItem{
					Threshold: 90,
					Icon:      "cpu",
				},
				Memory: CPUItem{
					Threshold: 90,
					Icon:      "speedometer",
				},
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
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
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
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

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
	_ = v.BindEnv("web.host", "LMON_WEB_HOST")
	_ = v.BindEnv("web.port", "LMON_WEB_PORT")
	_ = v.BindEnv("web.dashboard_title", "LMON_WEB_DASHBOARD_TITLE")

	// Monitoring config
	_ = v.BindEnv("monitoring.interval", "LMON_MONITORING_INTERVAL")

	// Webhook config
	_ = v.BindEnv("webhook.enabled", "LMON_WEBHOOK_ENABLED")
	_ = v.BindEnv("webhook.url", "LMON_WEBHOOK_URL")
}

// Save saves the configuration to a file
func Save(config *Config, path string) error {
	log.Printf("config.Save called with path: %s", path)
	v := viper.New()
	v.SetConfigType("yaml")

	// Set the config values
	err := v.MergeConfigMap(map[string]interface{}{
		"web": map[string]interface{}{
			"host":            config.Web.Host,
			"port":            config.Web.Port,
			"dashboard_title": config.Web.DashboardTitle,
		},
		"monitoring": config.Monitoring,
		"webhook":    config.Webhook,
	})
	if err != nil {
		log.Printf("config.Save merge error: %v", err)
		return fmt.Errorf("failed to merge config: %w", err)
	}

	// Marshal to YAML for debug
	yamlBytes, yamlErr := yaml.Marshal(config)
	if yamlErr != nil {
		log.Printf("config.Save marshal error: %v", yamlErr)
	} else {
		log.Printf("config.Save writing data:\n%s", string(yamlBytes))
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("config.Save mkdir error: %v", err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write config to file
	if err := v.WriteConfigAs(path); err != nil {
		log.Printf("config.Save write error: %v", err)
		return fmt.Errorf("failed to write config: %w", err)
	}

	log.Printf("config.Save completed for path: %s", path)
	return nil
}
