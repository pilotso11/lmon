package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/mapper"
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
Group=simple
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

	// subscribe to interrupts
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGKILL, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize logger
	log.SetOutput(os.Stdout)
	log.Println("Starting lmon - Lightweight Monitoring Service")

	// load config
	l := config.NewLoader("", nil)
	cfg, err := l.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// startup monitoring
	mon := monitors.NewService(ctx, time.Duration(cfg.Monitoring.Interval)*time.Second, time.Second, nil)

	// start server
	server, err := web.NewServerWithContext(ctx, cfg, l, mon, mapper.NewMapper(nil))
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// apply config
	err = server.SetConfig(ctx, cfg.Monitoring)
	if err != nil {
		log.Fatalf("Failed to set initial config: %v", err)
	}

	server.Start()

	// wait for interrupt
	<-ctx.Done()

	_ = server.Stop()

	log.Println("Shutdown complete")
}
