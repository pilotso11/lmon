package web

import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/mapper"
)

// Server encapsulates the HTTP server, configuration, and monitoring services.
type Server struct {
	mu         sync.Mutex        // Mutex to protect concurrent access to server state
	config     *config.Config    // Application configuration
	loader     *config.Loader    // Configuration loader
	monitor    *monitors.Service // Monitoring service
	httpServer *http.Server      // Underlying HTTP server
	ctx        context.Context   // Context for server lifecycle
	router     *http.ServeMux    // HTTP request router
	serverUrl  string            // URL where the server is accessible
	mapper     mapper.Mapper     // Config Mapper
	listener   net.Listener      // Network listener for incoming connections
}

// NewServerWithContext creates a new Server instance using the provided context, configuration,
// monitoring service, and optional provider implementations. If the port in the config is set to 0,
// the server will listen on a random available port.
func NewServerWithContext(ctx context.Context, cfg *config.Config, loader *config.Loader, monitorService *monitors.Service, builder mapper.Mapper) (*Server, error) {
	// Create router
	router := http.NewServeMux()

	// Create server
	server := &Server{
		config:  cfg,
		monitor: monitorService,
		router:  router,
		ctx:     ctx,
		mapper:  builder,
		loader:  loader,
	}

	addr := fmt.Sprintf("%s:%d", server.config.Web.Host, server.config.Web.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server.serverUrl = "http://" + ln.Addr().String()
	server.listener = ln

	server.httpServer = &http.Server{
		Addr:    ln.Addr().String(),
		Handler: router,
	}

	// Set up routes
	server.setupRoutes(ctx)

	// Automatically stop the server when the context is cancelled.
	go func() {
		<-ctx.Done()
		_ = server.Stop()
	}()

	return server, nil
}

// Start launches the web server in a separate goroutine.
// Returns immediately after starting the server.
func (s *Server) Start() error {
	log.Printf("Starting webserver on: %v", s.serverUrl)
	go func() {
		_ = s.httpServer.Serve(s.listener)
	}()
	return nil
}

// Stop gracefully shuts down the web server, waiting up to 5 seconds for active connections to close.
func (s *Server) Stop() error {
	if s.httpServer != nil {
		log.Printf("Stopping webserver")
		// Create a timeout context for shutdown
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// setupRoutes registers all HTTP endpoints and their handlers for the web server.
func (s *Server) setupRoutes(ctx context.Context) {
	// Serve static files (e.g., favicon)
	s.router.HandleFunc("GET /static/", s.handleStatic)

	// Health check endpoint for liveness/readiness probes
	s.router.HandleFunc("GET /healthz", s.handleHealthCheck)

	// Main dashboard and configuration pages
	s.router.HandleFunc("GET /", s.handleTemplate())
	s.router.HandleFunc("GET /index.html", s.handleTemplate())
	s.router.HandleFunc("GET /config", s.handleTemplate())

	// API endpoints for monitoring data and configuration management
	s.router.HandleFunc("GET /api/items", s.handleGetItems)
	s.router.HandleFunc("GET /api/items/{id}", s.handleGetItem)
	s.router.HandleFunc("GET /api/config", s.handleGetConfig)
	s.router.HandleFunc("POST /api/config/interval", s.handleIntervalUpdate(ctx))
	s.router.HandleFunc("POST /api/config/system", s.handleUpdateSystemConfig(ctx))
	s.router.HandleFunc("POST /api/config/disk", s.handleAddDiskMonitor)
	s.router.HandleFunc("POST /api/config/healthcheck", s.handleAddHealthCheck)
	s.router.HandleFunc("POST /api/config/webhook", s.handleUpdateWebhook)
	s.router.HandleFunc("DELETE /api/config/{type}/{id}", s.handleDeleteMonitor)
}

//go:embed static/*
var staticFS embed.FS

// handleStatic serves static files such as favicon or other assets under /static/.
// Currently only serves app.ico; returns 404 for other files.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, staticFS, r.URL.Path)
}

//go:embed templates/index.html
var indexHtml string

//go:embed templates/config.html
var configHtml string

// handleIndex returns an HTTP handler function that renders the main dashboard page.
// Uses the embedded index.html template and injects configuration values.
func (s *Server) handleTemplate() func(w http.ResponseWriter, r *http.Request) {
	// load templates
	tIndex, err := template.New("index.html").Parse(indexHtml)
	if err != nil {
		log.Printf("handleTemplate: %v", err)
	}

	tConfig, err := template.New("config.html").Parse(configHtml)
	if err != nil {
		log.Printf("handleTemplate: %v", err)
	}

	// handler
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		// if we failed to load ...
		if err != nil {
			log.Printf("handleIndex %v: %v", r.URL, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// template data
		data := map[string]any{
			"title":            s.config.Monitoring.System.Title,
			"dashboard_title":  s.config.Monitoring.System.Title,
			"refresh_interval": s.config.Monitoring.Interval,
		}

		// execute the template
		switch r.URL.Path {
		case "/", "/index.html":
			err = tIndex.ExecuteTemplate(w, "index.html", data)
		case "/config":
			err = tConfig.ExecuteTemplate(w, "config.html", data)
		default:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}

		if err != nil {
			log.Printf("handleTemplate %v: %v", r.URL, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// writeJson serializes the given data as JSON and writes it to the HTTP response.
// Sets the Content-Type header to application/json. Returns 500 on error.
func (s *Server) writeJson(w http.ResponseWriter, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("writeJson %v: %v", reflect.TypeOf(data), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Printf("writeJson %v: %v", reflect.TypeOf(data), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleGetItems responds with a JSON object containing all monitored items and their statuses.
func (s *Server) handleGetItems(w http.ResponseWriter, r *http.Request) {
	s.writeJson(w, s.monitor.Results())
}

// handleGetItem responds with a JSON object for a specific monitored item, identified by its ID.
// Returns 404 if the item is not found.
func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	items := s.monitor.Results()
	item, ok := items[id]
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		log.Printf("handleGetItem: item not found: %s", id)
		return
	}
	s.writeJson(w, item)
}

// handleGetConfig responds with the current server configuration as JSON.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeJson(w, s.config)
}

// unmarshallBody reads and unmarshals the JSON request body into the provided data structure.
// Returns true on success, or writes an error response and returns false on failure.
func (s *Server) unmarshallBody(w http.ResponseWriter, r *http.Request, data any) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("unmarshallBody %s: %v", r.URL, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		log.Printf("unmarshallBody %s: %v", r.URL, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

// handleUpdateSystemConfig processes a request to update the system configuration.
// Expects a JSON body with the new configuration.
func (s *Server) handleUpdateSystemConfig(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		var cfg config.SystemConfig
		ok := s.unmarshallBody(w, r, &cfg)
		if !ok {
			return
		}

		// Set title
		s.config.Monitoring.System.Title = cfg.Title

		// Apply cpu config
		cpu, err := s.mapper.NewCpu(ctx, cfg.CPU)
		if err != nil {
			log.Printf("handleUpdateSystemConfig (cpu): %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		err = s.monitor.Add(ctx, cpu)
		if err != nil {
			log.Printf("handleUpdateSystemConfig (cpu): %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		// Apply mem config
		mem, err := s.mapper.NewMem(ctx, cfg.Memory)
		if err != nil {
			log.Printf("handleUpdateSystemConfig (mem): %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		err = s.monitor.Add(ctx, mem)
		if err != nil {
			log.Printf("handleUpdateSystemConfig (mem): %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		s.saveConfig(w)
	}
}

func (s *Server) saveConfig(w http.ResponseWriter) {
	// Save config
	err := s.monitor.Save(s.config)
	if err != nil {
		log.Printf("handleUpdateSystemConfig (save): %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = s.loader.Save(s.config)
	if err != nil {
		log.Printf("handleUpdateSystemConfig (save): %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// done
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// handleIntervalUpdate processes an HTTP request to update the monitoring interval configuration dynamically.
func (s *Server) handleIntervalUpdate(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var cfg struct {
			Interval int
		}
		ok := s.unmarshallBody(w, r, &cfg)
		if !ok {
			return
		}
		s.monitor.SetPeriod(ctx, time.Duration(cfg.Interval)*time.Second, time.Second)
		s.saveConfig(w)
	}
}

// handleAddDiskMonitor processes a request to add a new disk monitor.
// Expects a JSON body describing the disk monitor to add.
func (s *Server) handleAddDiskMonitor(w http.ResponseWriter, r *http.Request) {
	// todo: implement me

}

// handleAddHealthCheck processes a request to add a new health check monitor.
// Expects a JSON body describing the health check to add.
func (s *Server) handleAddHealthCheck(w http.ResponseWriter, r *http.Request) {
	// todo: implement me

}

// handleUpdateWebhook processes a request to update the webhook configuration.
// Expects a JSON body with the new webhook settings.
func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	// todo: implement me

}

// handleDeleteMonitor processes a request to delete a monitor by type and ID.
func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	// todo: implement me
}

// handleHealthCheck responds with HTTP 200 OK for health check probes.
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}
