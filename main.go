package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/mapper"
	"lmon/systemd"
	"lmon/web"
)

func main() {
	// Parse command line flags
	installFlag := flag.Bool("install-service", false, "Install lmon as a systemd service")
	uninstallFlag := flag.Bool("uninstall-service", false, "Uninstall the lmon systemd service")
	flag.Parse()

	// Handle service installation/uninstallation
	if *installFlag {
		if err := systemd.InstallService(); err != nil {
			log.Fatalf("Failed to install service: %v", err)
		}
		return
	}

	if *uninstallFlag {
		if err := systemd.UninstallService(); err != nil {
			log.Fatalf("Failed to uninstall service: %v", err)
		}
		return
	}

	// subscribe to interrupts
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, os.Interrupt, syscall.SIGTERM)
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

	server.Start(ctx)

	// wait for interrupt
	<-ctx.Done()

	_ = server.Stop()

	log.Println("Shutdown complete")
}
