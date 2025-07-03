package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"lmon/config"
	"lmon/monitor"
	"lmon/web"
)

func main() {
	// Initialize logger
	log.SetOutput(os.Stdout)
	log.Println("Starting lmon - Lightweight Monitoring Service")

	// Create a context that will be canceled on termination signals
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize monitoring service with context
	monitoringService := monitor.NewServiceWithContext(ctx, cfg)

	// Start monitoring routines
	monitoringService.Start()

	// Initialize web server with context
	webServer := web.NewServerWithContext(ctx, cfg, monitoringService)

	// Start web server in a goroutine
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Web.Host, cfg.Web.Port)
		log.Printf("Starting web server on %s", addr)
		if err := webServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	// Wait for context to be canceled (when a termination signal is received)
	<-ctx.Done()

	log.Println("Shutting down...")

	// Stop receiving signals to prevent additional signal notifications
	stop()

	// Wait for graceful shutdown
	webServer.Stop()

	log.Println("Shutdown complete")
}
