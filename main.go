package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"lmon/config"
	"lmon/monitor"
	"lmon/web"
)

// installService installs lmon as a systemd service
func installService() error {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create the target directory
	targetDir := "/opt/lmon"
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Copy the binary to the target location
	targetPath := filepath.Join(targetDir, "lmon")
	sourceFile, err := os.Open(execPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func(sourceFile *os.File) {
		_ = sourceFile.Close()
	}(sourceFile)

	targetFile, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer func(targetFile *os.File) {
		_ = targetFile.Close()
	}(targetFile)

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Copy the service file
	serviceFilePath := filepath.Join(filepath.Dir(execPath), "lmon.service")
	if _, err := os.Stat(serviceFilePath); os.IsNotExist(err) {
		// If the service file doesn't exist in the same directory as the executable,
		// use the embedded service file content
		serviceContent := `[Unit]
Description=lmon - Lightweight Monitoring Service
After=network.target

[Service]
Type=simple
User=lmon
Group=lmon
WorkingDirectory=/opt/lmon
ExecStart=/opt/lmon/lmon
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Environment variables for configuration
Environment=LMON_WEB_HOST=0.0.0.0
Environment=LMON_WEB_PORT=8080

[Install]
WantedBy=multi-user.target
`
		if err := os.WriteFile("/etc/systemd/system/lmon.service", []byte(serviceContent), 0644); err != nil {
			return fmt.Errorf("failed to write service file: %w", err)
		}
	} else {
		// Copy the existing service file
		serviceFile, err := os.Open(serviceFilePath)
		if err != nil {
			return fmt.Errorf("failed to open service file: %w", err)
		}
		defer func(serviceFile *os.File) {
			_ = serviceFile.Close()
		}(serviceFile)

		targetServiceFile, err := os.OpenFile("/etc/systemd/system/lmon.service", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to create target service file: %w", err)
		}
		defer func(targetServiceFile *os.File) {
			_ = targetServiceFile.Close()
		}(targetServiceFile)

		if _, err := io.Copy(targetServiceFile, serviceFile); err != nil {
			return fmt.Errorf("failed to copy service file: %w", err)
		}
	}

	// Create the lmon user and group if they don't exist
	// This is a simple check; in a real-world scenario, you might want to use the user package
	// or exec the useradd command directly
	if _, err := os.Stat("/etc/passwd"); err == nil {
		// Execute the useradd command
		cmd := exec.Command("useradd", "-r", "-s", "/bin/false", "lmon")
		if err := cmd.Run(); err != nil {
			log.Printf("Note: Failed to create user lmon (it may already exist): %v", err)
		}
	}

	// Create the configuration directory
	if err := os.MkdirAll("/etc/lmon", 0755); err != nil {
		return fmt.Errorf("failed to create configuration directory: %w", err)
	}

	// Enable the service
	cmd := exec.Command("systemctl", "enable", "lmon")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	log.Println("Service installed successfully. You can start it with: sudo systemctl start lmon")
	return nil
}

// uninstallService removes the lmon systemd service
func uninstallService() error {
	// Stop the service if it's running
	stopCmd := exec.Command("systemctl", "stop", "lmon")
	if err := stopCmd.Run(); err != nil {
		log.Printf("Warning: Failed to stop service (it may not be running): %v", err)
	}

	// Disable the service
	disableCmd := exec.Command("systemctl", "disable", "lmon")
	if err := disableCmd.Run(); err != nil {
		log.Printf("Warning: Failed to disable service: %v", err)
	}

	// Remove the service file
	if err := os.Remove("/etc/systemd/system/lmon.service"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	log.Println("Service uninstalled successfully.")
	return nil
}

func main() {
	// Parse command line flags
	installFlag := flag.Bool("install-service", false, "Install lmon as a systemd service")
	uninstallFlag := flag.Bool("uninstall-service", false, "Uninstall the lmon systemd service")
	flag.Parse()

	// Handle service installation/uninstallation
	if *installFlag {
		if err := installService(); err != nil {
			log.Fatalf("Failed to install service: %v", err)
		}
		return
	}

	if *uninstallFlag {
		if err := uninstallService(); err != nil {
			log.Fatalf("Failed to uninstall service: %v", err)
		}
		return
	}

	// Initialize logger
	log.SetOutput(os.Stdout)
	log.Println("Starting lmon - Lightweight Monitoring Service")

	// Create a context that will be canceled on termination signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// In test mode, use a temp config file to isolate config changes
	var cfg *config.Config
	var err error
	testMode := os.Getenv("LMON_TEST_MODE") == "1"
	if testMode {
		tmpConfigPath := os.Getenv("LMON_TEST_CONFIG_PATH")
		if tmpConfigPath == "" {
			tmpConfigPath = "./lmon-test-config.yaml"
		}
		log.Printf("LMON_TEST_MODE=1: Using temp config file: %s", tmpConfigPath)
		// If the file doesn't exist, create it with defaults
		if _, statErr := os.Stat(tmpConfigPath); os.IsNotExist(statErr) {
			defaultCfg := config.DefaultConfig()
			if saveErr := config.Save(defaultCfg, tmpConfigPath); saveErr != nil {
				log.Fatalf("Failed to create temp config file: %v", saveErr)
			}
			log.Printf("Created temp config file at: %s", tmpConfigPath)
		} else {
			log.Printf("Temp config file already exists at: %s", tmpConfigPath)
		}
		cfg, err = config.LoadFromFile(tmpConfigPath)
		if err != nil {
			log.Fatalf("Failed to load temp config: %v", err)
		}
		// Patch the config path in the web server so all saves go to the temp file
		os.Setenv("LMON_CONFIG_PATH", tmpConfigPath)
	} else {
		cfg, err = config.Load()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	}

	// Log the loaded configuration for debugging
	log.Printf("Loaded configuration: Web: {Host: %s, Port: %d}", cfg.Web.Host, cfg.Web.Port)
	log.Printf("Monitoring: {Interval: %d}", cfg.Monitoring.Interval)
	log.Printf("System: {CPUThreshold: %d, MemoryThreshold: %d, CPUIcon: %s, MemoryIcon: %s}",
		cfg.Monitoring.System.CPU.Threshold,
		cfg.Monitoring.System.Memory.Threshold,
		cfg.Monitoring.System.CPU.Icon,
		cfg.Monitoring.System.Memory.Icon)
	log.Printf("Disk monitors: %d", len(cfg.Monitoring.Disk))
	for i, disk := range cfg.Monitoring.Disk {
		log.Printf("  Disk[%d]: {Path: %s, Threshold: %d, Icon: %s}", i, disk.Path, disk.Threshold, disk.Icon)
	}
	log.Printf("Health checks: %d", len(cfg.Monitoring.Healthchecks))
	for i, health := range cfg.Monitoring.Healthchecks {
		log.Printf("  Health[%d]: {Name: %s, URL: %s, Interval: %d, Timeout: %d, Icon: %s}",
			i, health.Name, health.URL, health.Interval, health.Timeout, health.Icon)
	}
	log.Printf("Webhook: {Enabled: %t, URL: %s}", cfg.Webhook.Enabled, cfg.Webhook.URL)

	// Use real monitors with mock providers in test mode for integration tests
	var monitoringService *monitor.Service
	if os.Getenv("LMON_TEST_MODE") == "1" {
		log.Println("LMON_TEST_MODE=1: Using real monitors with mock providers for integration tests")
		monitoringService = monitor.NewServiceWithMonitors(
			cfg,
			monitor.NewDiskMonitorWithProvider(cfg, &monitor.AlwaysHealthyDiskUsageProvider{}),
			monitor.NewSystemMonitor(cfg), // Optionally, you can mock CPU/mem providers too
			monitor.NewHealthMonitor(cfg), // Optionally, you can mock HTTP client
			&monitor.AlwaysNoopWebhookSender{},
		)
	} else {
		// Initialize monitoring service with context
		monitoringService = monitor.NewServiceWithContext(ctx, cfg)
	}

	// Start monitoring routines
	monitoringService.Start()

	// Initialize web server with context and correct config path
	var configPath string
	if testMode {
		configPath = os.Getenv("LMON_TEST_CONFIG_PATH")
		if configPath == "" {
			configPath = "./lmon-test-config.yaml"
		}
	} else {
		configPath = "../config.yaml"
	}
	webServer := web.NewServerWithContext(ctx, cfg, monitoringService, configPath)

	// Start web server in a goroutine
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Web.Host, cfg.Web.Port)
		log.Printf("Starting web server on %s", addr)
		if err := webServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	// Wait for context to be canceled (when a termination signal is received)
	<-ctx.Done()

	log.Println("Shutting down...")

	// Stop receiving signals to prevent additional signal notifications
	stop()

	// Wait for graceful shutdown
	_ = webServer.Stop()

	log.Println("Shutdown complete")
}
