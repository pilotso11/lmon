package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lmon/common"
	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/mapper"
	"lmon/web"
)

var logWriter *common.AtomicWriter

func main() {
	// subscribe to interrupts
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize logger
	if logWriter != nil {
		// for testing, redirect stdout to buffer
		log.SetOutput(logWriter)
	} else {
		log.SetOutput(os.Stdout)
	}
	log.Println("Starting lmon - Lightweight Monitoring Service")

	// load config
	l := config.NewLoader("", nil)
	cfg, err := l.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// startup monitoring
	mon := monitors.NewService(ctx, time.Duration(cfg.Monitoring.Interval)*time.Second, 10*time.Second, nil)

	// Create mapper with allowed restart containers
	m := mapper.NewMapper(nil)
	m.AllowedRestartContainers = cfg.Monitoring.AllowedRestartContainers

	// start server
	server, err := web.NewServerWithContext(ctx, cfg, l, mon, m)
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
