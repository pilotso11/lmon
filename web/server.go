package web

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"lmon/config"
)

// Server represents the web server
type Server struct {
	config     *config.Config
	monitor    MonitorServiceInterface
	router     *gin.Engine
	httpServer *http.Server
	ctx        context.Context
}

// NewServer creates a new web server
func NewServer(cfg *config.Config, monitorService MonitorServiceInterface) *Server {
	return NewServerWithContext(context.Background(), cfg, monitorService)
}

// NewServerWithContext creates a new web server with the provided context
func NewServerWithContext(ctx context.Context, cfg *config.Config, monitorService MonitorServiceInterface) *Server {
	// Create gin router
	router := gin.Default()

	// Load HTML templates from embedded filesystem
	templ, err := GetTemplatesFSWithRoot()
	if err != nil {
		panic(err)
	}
	router.SetHTMLTemplate(template.Must(template.New("").ParseFS(templ, "*.html")))

	// Create server
	server := &Server{
		config:  cfg,
		monitor: monitorService,
		router:  router,
		ctx:     ctx,
	}

	// Set up routes
	server.setupRoutes()

	return server
}

// Start starts the web server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Web.Host, s.config.Web.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	return s.httpServer.ListenAndServe()
}

// Stop stops the web server
func (s *Server) Stop() error {
	if s.httpServer != nil {
		// Create a timeout context for shutdown
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Serve static files
	s.router.Static("/static", "./web/static")

	// API routes
	api := s.router.Group("/api")
	{
		// Get all items
		api.GET("/items", s.handleGetItems)

		// Get a specific item
		api.GET("/items/:id", s.handleGetItem)

		// Get configuration
		api.GET("/config", s.handleGetConfig)

		// Update configuration
		api.POST("/config", s.handleUpdateConfig)

		// Add disk monitor
		api.POST("/config/disk", s.handleAddDiskMonitor)

		// Add health check monitor
		api.POST("/config/healthcheck", s.handleAddHealthCheck)

		// Update webhook configuration
		api.POST("/config/webhook", s.handleUpdateWebhook)

		// Delete a monitor
		api.DELETE("/config/:type/:id", s.handleDeleteMonitor)
	}

	// Health check endpoint
	s.router.GET("/healthz", s.handleHealthCheck)

	// Web UI routes
	s.router.GET("/", s.handleIndex)
	s.router.GET("/config", s.handleConfigPage)
}

// handleIndex handles the index page request
func (s *Server) handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":            "lmon - Lightweight Monitoring",
		"dashboard_title":  s.config.Web.DashboardTitle,
		"refresh_interval": s.config.Monitoring.Interval,
	})
}

// handleConfigPage handles the configuration page request
func (s *Server) handleConfigPage(c *gin.Context) {
	c.HTML(http.StatusOK, "config.html", gin.H{
		"title":           "lmon - Configuration",
		"dashboard_title": s.config.Web.DashboardTitle,
	})
}

// handleGetItems handles the request to get all monitored items
func (s *Server) handleGetItems(c *gin.Context) {
	items := s.monitor.GetItems()
	c.JSON(http.StatusOK, items)
}

// handleGetItem handles the request to get a specific monitored item
func (s *Server) handleGetItem(c *gin.Context) {
	id := c.Param("id")
	item := s.monitor.GetItem(id)
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// handleGetConfig handles the request to get the configuration
func (s *Server) handleGetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, s.config)
}

// handleUpdateConfig handles the request to update the configuration
func (s *Server) handleUpdateConfig(c *gin.Context) {
	var cfg config.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid configuration: %v", err)})
		return
	}

	// Update the server's in-memory configuration
	s.config = &cfg

	// Save configuration
	if err := config.Save(s.config, "config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save configuration: %v", err)})
		return
	}

	// Trigger immediate refresh of all checks to apply new thresholds
	s.monitor.RefreshChecks()

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
}

// handleAddDiskMonitor handles the request to add a disk monitor
func (s *Server) handleAddDiskMonitor(c *gin.Context) {
	var diskCfg config.DiskConfig
	if err := c.ShouldBindJSON(&diskCfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid disk configuration: %v", err)})
		return
	}

	// Add disk monitor to configuration
	s.config.Monitoring.Disk = append(s.config.Monitoring.Disk, diskCfg)

	// Save configuration
	if err := config.Save(s.config, "config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save configuration: %v", err)})
		return
	}

	// Trigger immediate refresh of all checks to apply new disk monitor
	s.monitor.RefreshChecks()

	c.JSON(http.StatusOK, gin.H{"message": "Disk monitor added successfully"})
}

// handleAddHealthCheck handles the request to add a health check
func (s *Server) handleAddHealthCheck(c *gin.Context) {
	var healthCfg config.HealthcheckConfig
	if err := c.ShouldBindJSON(&healthCfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid health check configuration: %v", err)})
		return
	}

	// Add health check to configuration
	s.config.Monitoring.Healthchecks = append(s.config.Monitoring.Healthchecks, healthCfg)

	// Save configuration
	if err := config.Save(s.config, "config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save configuration: %v", err)})
		return
	}

	// Trigger immediate refresh of all checks to apply new health check
	s.monitor.RefreshChecks()

	c.JSON(http.StatusOK, gin.H{"message": "Health check added successfully"})
}

// handleUpdateWebhook handles the request to update webhook configuration
func (s *Server) handleUpdateWebhook(c *gin.Context) {
	var webhookCfg config.WebhookConfig
	if err := c.ShouldBindJSON(&webhookCfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid webhook configuration: %v", err)})
		return
	}

	// Update webhook configuration
	s.config.Webhook = webhookCfg

	// Save configuration
	if err := config.Save(s.config, "config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save configuration: %v", err)})
		return
	}

	// Trigger immediate refresh of all checks to apply new webhook configuration
	s.monitor.RefreshChecks()

	c.JSON(http.StatusOK, gin.H{"message": "Webhook configuration updated successfully"})
}

// handleDeleteMonitor handles the request to delete a monitor
func (s *Server) handleDeleteMonitor(c *gin.Context) {
	monitorType := c.Param("type")
	monitorID := c.Param("id")

	switch monitorType {
	case "disk":
		// Find and remove disk monitor
		for i, disk := range s.config.Monitoring.Disk {
			if disk.Path == monitorID {
				s.config.Monitoring.Disk = append(s.config.Monitoring.Disk[:i], s.config.Monitoring.Disk[i+1:]...)
				break
			}
		}
	case "healthcheck":
		// Find and remove health check
		for i, health := range s.config.Monitoring.Healthchecks {
			if health.Name == monitorID {
				s.config.Monitoring.Healthchecks = append(s.config.Monitoring.Healthchecks[:i], s.config.Monitoring.Healthchecks[i+1:]...)
				break
			}
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Unknown monitor type: %s", monitorType)})
		return
	}

	// Save configuration
	if err := config.Save(s.config, "config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save configuration: %v", err)})
		return
	}

	// Trigger immediate refresh of all checks to apply monitor deletion
	s.monitor.RefreshChecks()

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s monitor deleted successfully", monitorType)})
}

// handleHealthCheck handles the health check endpoint request
func (s *Server) handleHealthCheck(c *gin.Context) {
	// Get all monitored items to check overall system health
	items := s.monitor.GetItems()

	// Check if any item is in critical status
	hasCritical := false
	for _, item := range items {
		if item.Status == "CRITICAL" {
			hasCritical = true
			break
		}
	}

	// If any item is critical, return 503 Service Unavailable
	// Otherwise return 200 OK
	if hasCritical {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": "One or more monitored items are in CRITICAL state",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"message":   "All systems operational",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0", // You might want to get this from a version constant
	})
}
