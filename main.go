package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lmon/aggregator"
	"lmon/common"
	"lmon/config"
	"lmon/db"
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

	// Resolve operating mode
	mode := config.ResolveMode()
	log.Printf("Operating mode: %s", mode)

	switch mode {
	case config.ModeAggregator:
		startAggregator(ctx, cfg, l)
	default:
		startNode(ctx, cfg, l)
	}
}

func startNode(ctx context.Context, cfg *config.Config, l *config.Loader) {
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

	// Wire up database if configured
	store, writer, retention := setupDatabase(ctx, cfg)
	if store != nil {
		server.SetStore(store)
	}
	if writer != nil {
		server.SetWriter(writer)
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
	if retention != nil {
		retention.Stop()
	}
	if writer != nil {
		writer.Close()
	}
	if store != nil {
		_ = store.Close()
	}

	log.Println("Shutdown complete")
}

// setupDatabase creates the database store, buffered writer, and retention manager
// if a database URL is configured. Returns nil values if no database is configured.
func setupDatabase(ctx context.Context, cfg *config.Config) (db.Store, *db.BufferedWriter, *db.RetentionManager) {
	if cfg.Database.URL == "" {
		return nil, nil, nil
	}

	store, err := db.NewPostgresStore(ctx, cfg.Database.URL)
	if err != nil {
		log.Printf("Database setup failed: %v", err)
		return nil, nil, nil
	}

	writeInterval := time.Duration(cfg.Database.WriteInterval) * time.Second
	writer := db.NewBufferedWriter(store, cfg.Database.BatchSize, writeInterval)

	retention := db.NewRetentionManager(
		store,
		cfg.Database.RetentionDays,
		cfg.Database.BatchSize,
		cfg.Database.PruneInterval,
		cfg.Database.CompactAfter,
		cfg.Database.CompactInterval,
	)
	retention.Start(ctx)

	log.Printf("Database persistence enabled (retention=%dd, compact_after=%dm, compact_interval=%dm)",
		cfg.Database.RetentionDays, cfg.Database.CompactAfter, cfg.Database.CompactInterval)

	return store, writer, retention
}

func startAggregator(ctx context.Context, cfg *config.Config, l *config.Loader) {
	log.Printf("Starting aggregator mode")

	// startup monitoring service for local monitors (k8s events, nodes, etc.)
	mon := monitors.NewService(ctx, time.Duration(cfg.Monitoring.Interval)*time.Second, 10*time.Second, nil)

	// Create mapper
	m := mapper.NewMapper(nil)

	// start server
	server, err := web.NewServerWithContext(ctx, cfg, l, mon, m)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Wire up database if configured
	store, writer, retention := setupDatabase(ctx, cfg)
	if store != nil {
		server.SetStore(store)
	}
	if writer != nil {
		server.SetWriter(writer)
	}

	// apply config (will include k8s monitors if kubernetes.enabled)
	err = server.SetConfig(ctx, cfg.Monitoring)
	if err != nil {
		log.Fatalf("Failed to set initial config: %v", err)
	}

	// Create aggregator provider.
	// TODO: When running in a real Kubernetes cluster, use aggregator.NewK8sProvider()
	// with a real PodLister backed by client-go. The mock provider is a no-op placeholder.
	var provider aggregator.Provider
	log.Printf("WARNING: Aggregator using mock provider. For Kubernetes cluster discovery, " +
		"configure a K8sProvider with client-go PodLister.")
	provider = aggregator.NewMockProvider(nil, nil)

	agg := aggregator.NewAggregator(
		provider,
		cfg.Aggregator.NodeLabel,
		cfg.Aggregator.NodePort,
		cfg.Aggregator.NodeMetricsPath,
		time.Duration(cfg.Aggregator.ScrapeInterval)*time.Second,
		nil,
	)

	// Setup aggregator routes (overrides default dashboard with aggregator view)
	server.SetupAggregatorRoutes(agg)

	agg.Start(ctx)
	server.Start(ctx)

	// wait for interrupt
	<-ctx.Done()

	agg.Stop()
	_ = server.Stop()
	if retention != nil {
		retention.Stop()
	}
	if writer != nil {
		writer.Close()
	}
	if store != nil {
		_ = store.Close()
	}

	log.Println("Aggregator shutdown complete")
}
