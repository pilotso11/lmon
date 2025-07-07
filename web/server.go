package web

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
	"lmon/monitors/system"
)

type Implementations struct {
	disk   disk.UsageProvider
	health healthcheck.UsageProvider
	cpu    system.CpuProvider
	mem    system.MemProvider
}

// DefaultImplementations are all nil implementations, with no testing overrides.
func DefaultImplementations() Implementations {
	return Implementations{}
}

// Server represents the web server
type Server struct {
	mu         sync.Mutex
	config     *config.Config
	monitor    *monitors.Service
	httpServer *http.Server
	ctx        context.Context
	router     *http.ServeMux
	serverUrl  string
	impls      Implementations
	listener   net.Listener
}

// NewServerWithContext creates a new web server with the provided context
// To run on a random port specify port as 0 in the config or env.
func NewServerWithContext(ctx context.Context, cfg *config.Config, monitorService *monitors.Service, impl Implementations) (*Server, error) {
	// Create router
	router := http.NewServeMux()

	// Create server
	server := &Server{
		config:  cfg,
		monitor: monitorService,
		router:  router,
		ctx:     ctx,
		impls:   impl,
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
	server.setupRoutes()

	go func() {
		<-ctx.Done()
		_ = server.Stop()
	}()

	return server, nil
}

// Start starts the web server
func (s *Server) Start() error {
	log.Printf("Starting webserver on: %v", s.serverUrl)
	go func() {
		_ = s.httpServer.Serve(s.listener)
	}()
	return nil
}

// Stop stops the web server
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

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Serve static files
	s.router.HandleFunc("GET /static/", s.handleStatic)

	// my heealthcheck
	s.router.HandleFunc("GET /healthz", s.handleHealthCheck)

	s.router.HandleFunc("GET /", s.handleIndex())
	s.router.HandleFunc("GET /index.html", s.handleIndex())
	s.router.HandleFunc("GET /config", s.handleConfigPage())

	// API routes
	s.router.HandleFunc("GET /api/items", s.handleGetItems)
	s.router.HandleFunc("GET /api/items/{id}", s.handleGetItem)
	s.router.HandleFunc("GET /api/config", s.handleGetConfig)
	s.router.HandleFunc("POST /api/config/system", s.handleUpdateSystemConfig)
	s.router.HandleFunc("POST /api/config/disk", s.handleAddDiskMonitor)
	s.router.HandleFunc("POST /api/config/healthcheck", s.handleAddHealthCheck)
	s.router.HandleFunc("POST /api/config/webhook", s.handleUpdateWebhook)
	s.router.HandleFunc("DELETE /api/config/{type}/{id}", s.handleDeleteMonitor)
}

// handleStatic handles the static file request
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	switch strings.Replace(r.URL.Path, "/static/", "", -1) {
	case "app.ico":
		http.ServeFile(w, r, "static/app.ico")
	}
	log.Printf("handleStatic %v: not found", r.URL)
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

//go:embed templates/index.html
var indexHtml string

//go:embed templates/config.html
var configHtml string

// handleIndex handles the index page request
func (s *Server) handleIndex() func(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("index.html").Parse(indexHtml)
	if err != nil {
		log.Printf("index.html: %v", err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		if err != nil {
			log.Printf("handleIndex %v: %v", r.URL, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = t.ExecuteTemplate(w, "index.html",
			map[string]any{
				"title":            s.config.Monitoring.System.Title,
				"dashboard_title":  s.config.Monitoring.System.Title,
				"refresh_interval": s.config.Monitoring.Interval,
			})
		if err != nil {
			log.Printf("handleIndex %v: %v", r.URL, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// handleConfigPage handles the configuration page request
func (s *Server) handleConfigPage() func(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("config.html").Parse(configHtml)
	if err != nil {
		log.Printf("config.html: %v", err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		if err != nil {
			log.Printf("handleConfigPage %v: %v", r.URL, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = t.ExecuteTemplate(w, "config.html",
			map[string]any{
				"title":            s.config.Monitoring.System.Title,
				"dashboard_title":  s.config.Monitoring.System.Title,
				"refresh_interval": s.config.Monitoring.Interval,
			})
		if err != nil {
			log.Printf("handleConfigPage %v: %v", r.URL, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) writeJson(w http.ResponseWriter, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("writeJson %v: %v", reflect.TypeOf(data), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Printf("writeJson %v: %v", reflect.TypeOf(data), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

// handleGetItems handles the request to get all monitored items
func (s *Server) handleGetItems(w http.ResponseWriter, r *http.Request) {
	s.writeJson(w, s.monitor.Results())
}

// handleGetItem handles the request to get a specific monitored item
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

// handleGetConfig handles the request to get the configuration
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeJson(w, s.config)
}

func (s *Server) unmarshallBody(w http.ResponseWriter, r *http.Request, data any) bool {
	var body []byte
	n, err := r.Body.Read(body)
	if err != nil {
		log.Printf("unmarshallBody %s: %v", r.URL, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if n == 0 {
		log.Printf("unmarshallBody %s: %v", r.URL, err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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

// handleUpdateSystemConfig handles the request to update the system configuration
func (s *Server) handleUpdateSystemConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.SystemConfig
	ok := s.unmarshallBody(w, r, &cfg)
	if !ok {
		return
	}
	// todo: handle config update
}

// handleAddDiskMonitor handles the request to add a disk monitor
func (s *Server) handleAddDiskMonitor(w http.ResponseWriter, r *http.Request) {
	// todo: implement me

}

// handleAddHealthCheck handles the request to add a health check
func (s *Server) handleAddHealthCheck(w http.ResponseWriter, r *http.Request) {
	// todo: implement me

}

// handleUpdateWebhook handles the request to update webhook configuration
func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	// todo: implement me

}

// handleDeleteMonitor handles the request to delete a monitor
func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	// todo: implement me
}

// handleHealthCheck handles the health check endpoint request
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
