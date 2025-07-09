package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/mapper"
	"lmon/web"
)

func main() {
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
	mon := monitors.NewService(ctx, time.Duration(cfg.Monitoring.Interval)*time.Second, 10*time.Second, nil)

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
