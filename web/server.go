// Package web provides the HTTP server and web interface for lmon,
// including API endpoints for monitoring, configuration, and static content.
package web

import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
	"lmon/monitors/mapper"
	"lmon/monitors/system"
	"lmon/webhook"
)

type IconItem struct {
	Name string // Name of the Icon, used for display and identification.
	Icon string // Icon class Name, e.g., "bi-cpu" for Bootstrap Icons.
}

func defaultIconList() []IconItem {
	return []IconItem{
		{Name: "cpu", Icon: "bi-cpu"},
		{Name: "memory", Icon: "bi-memory"},
		{Name: "sd-card", Icon: "bi-sd-card"},
		{Name: "hdd", Icon: "bi-hdd"},
		{Name: "hdd-network", Icon: "bi-hdd-network"},
		{Name: "hdd-rack", Icon: "bi-hdd-rack"},
		{Name: "device-hdd", Icon: "bi-device-hdd"},
		{Name: "database", Icon: "bi-database"},
		{Name: "pc-horizontal", Icon: "bi-pc-horizontal"},
		{Name: "pc", Icon: "bi-pc"},
		{Name: "activity", Icon: "bi-activity"},
		{Name: "heart-pulse", Icon: "bi-heart-pulse"},
		{Name: "speedometer", Icon: "bi-speedometer"},
		{Name: "speedometer2", Icon: "bi-speedometer2"},
		{Name: "bar-chart", Icon: "bi-bar-chart"},
		{Name: "graph-up", Icon: "bi-graph-up"},
		{Name: "router", Icon: "bi-router"},
		{Name: "wifi", Icon: "bi-wifi"},
		{Name: "house", Icon: "bi-house"},
		{Name: "building", Icon: "bi-building"},
		{Name: "lightning", Icon: "bi-lightning"},
		{Name: "lightbulb", Icon: "bi-lightbulb"},
		{Name: "lamp", Icon: "bi-lamp"},
		{Name: "at", Icon: "bi-at"},
		{Name: "battery", Icon: "bi-battery"},
		{Name: "globe", Icon: "bi-globe"},
		{Name: "printer", Icon: "bi-printer"},
		{Name: "folder", Icon: "bi-folder"},
		{Name: "shield", Icon: "bi-shield"},
		{Name: "collection", Icon: "bi-collection"},
		{Name: "envelope", Icon: "bi-envelope"},
		{Name: "inbox", Icon: "bi-inbox"},
		{Name: "people", Icon: "bi-people"},
		{Name: "person-circle", Icon: "bi-person-circle"},
		{Name: "webcam", Icon: "bi-webcam"},
		{Name: "volume-up", Icon: "bi-volume-up"},
		{Name: "voicemail", Icon: "bi-voicemail"},
		{Name: "tv", Icon: "bi-tv"},
	}
}

type customResponseWriter struct {
	writer http.ResponseWriter // Pointer to the HTTP response writer for writing responses.
	code   int
}

func (w *customResponseWriter) Write(data []byte) (int, error) {
	if w.code != http.StatusNotFound {
		return w.writer.Write(data)
	}
	// fake it if we're returning an error
	return len(data), nil
}

func (w *customResponseWriter) WriteHeader(status int) {
	w.code = status
	if w.code != http.StatusNotFound {
		w.writer.WriteHeader(status)
		return
	}
}

func (w *customResponseWriter) Header() http.Header {
	return w.writer.Header()
}

// Server encapsulates the HTTP server, configuration, and monitoring services.
// It manages the lifecycle of the web server, routes, and provides thread-safe access to configuration.
type Server struct {
	mu         sync.Mutex        // Mutex to protect concurrent access to config.
	config     *config.Config    // Application configuration.
	loader     *config.Loader    // Configuration loader for persisting config changes.
	monitor    *monitors.Service // Monitoring service for system and custom checks.
	httpServer *http.Server      // Underlying HTTP server instance.
	ctx        context.Context   // Context for server lifecycle and cancellation.
	router     *http.ServeMux    // HTTP request router.
	ServerUrl  string            // URL where the server is accessible.
	mapper     mapper.Mapper     // Mapper for creating monitor implementations from config.
	listener   net.Listener      // Network listener for incoming connections.
}

// NewServerWithContext creates a new Server instance using the provided context, configuration,
// configuration loader, monitoring service, and mapper implementation.
// If the port in the config is set to 0, the server will listen on a random available port.
// Returns the initialized Server or an error if setup fails.
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
	server.ServerUrl = "http://" + ln.Addr().String()
	server.listener = ln

	server.httpServer = &http.Server{
		Addr:    ln.Addr().String(),
		Handler: LoggingHandler(router),
	}

	// Setup push to webhook callback
	server.monitor.SetPush(server.pushToWebhook)

	// Setup routes
	server.setupRoutes(ctx)

	err = server.SetConfig(ctx, cfg.Monitoring)
	if err != nil {
		log.Printf("NewServerWithContext: %v", err)
		return nil, fmt.Errorf("failed to set initial configuration: %w", err)
	}

	// Automatically stop the server when the context is cancelled.
	go func() {
		<-ctx.Done()
		log.Printf("Shutting down")
	}()

	return server, nil
}

func LoggingHandler(router *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Log the request

		cw := customResponseWriter{writer: w}
		// Call the actual handler
		router.ServeHTTP(&cw, r)

		if cw.code == http.StatusNotFound {
			http.Redirect(w, r, "/static/404.html", http.StatusFound)
		}

		elapsedTime := time.Since(start)
		log.Printf("[%v] (%d) %s %s %s %v", start.Format("2006-01-02 15:04:05.000"), cw.code, r.Method, r.URL.Path, r.RemoteAddr, elapsedTime)
	})
}

// Start launches the web server in a separate goroutine.
// Returns immediately after starting the server. The server will listen for incoming HTTP requests.
func (s *Server) Start(ctx context.Context) {
	rootUrl := "http://[::]"
	url := strings.Replace(s.ServerUrl, rootUrl, "http://localhost", 1)

	log.Printf("Starting webserver on: %v", url)
	go func() {
		_ = s.httpServer.Serve(s.listener)
	}()

	go func() {
		<-ctx.Done()
		log.Printf("Webserver stopping")
		if err := s.Stop(); err != nil {
			log.Printf("Error stopping webserver: %v", err)
		}
	}()
}

// Stop gracefully shuts down the web server, waiting up to 5 seconds for active connections to close.
// Returns an error if shutdown fails.
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
// This includes static assets, health checks, dashboard pages, and API endpoints for monitoring and configuration.
func (s *Server) setupRoutes(ctx context.Context) {

	templateHandler := s.handleTemplate()

	// Serve static files (e.g., favicon)
	s.router.HandleFunc("GET /static/", s.handleStatic)

	// Health check endpoint for liveness/readiness probes
	s.router.HandleFunc("GET /healthz", s.handleHealthCheck)

	// Endpoint to test webhook integration
	s.router.HandleFunc("POST /testhook", s.handleTestWebhook(ctx))

	// Main dashboard and configuration pages
	s.router.HandleFunc("GET /", templateHandler)
	s.router.HandleFunc("GET /index.html", templateHandler)
	s.router.HandleFunc("GET /config", templateHandler)
	s.router.HandleFunc("GET /mobile", templateHandler)

	// API endpoints for monitoring data and configuration management
	s.router.HandleFunc("GET /api/items", s.handleGetItems)
	s.router.HandleFunc("GET /api/items/{group}/{id}", s.handleGetItem)
	s.router.HandleFunc("GET /api/config", s.handleGetConfig)
	s.router.HandleFunc("POST /api/config/interval", s.handleIntervalUpdate(ctx))
	s.router.HandleFunc("POST /api/config/system", s.handleUpdateSystemConfig(ctx))
	s.router.HandleFunc("POST /api/config/disk/{id}", s.handleAddDiskMonitor(ctx))
	s.router.HandleFunc("POST /api/config/health/{id}", s.handleAddHealthCheck(ctx))
	s.router.HandleFunc("POST /api/config/ping/{id}", s.handleAddPingMonitor(ctx))
	s.router.HandleFunc("POST /api/config/webhook", s.handleUpdateWebhook(ctx))
	s.router.HandleFunc("DELETE /api/config/{type}/{id}", s.handleDeleteMonitor(ctx))
}

// getResultArrow returns a unicode arrow indicating the trend of a monitor's status change.
func getResultArrow(prev monitors.Result, result monitors.Result) any {
	if prev.Status == monitors.RAGUnknown {
		return ""
	}
	if result.Status > prev.Status {
		return "\u2197" // trending up in health
	} else {
		return "\u2198" // trending down in health
	}
}

// pushToWebhook asynchronously sends a notification to the configured webhook URL
// when a monitor's result changes. The payload includes a summary message.
func (s *Server) pushToWebhook(ctx context.Context, m monitors.Monitor, prev monitors.Result, result monitors.Result) {
	// Do this in a goroutine because we need to relock to access config.
	s.mu.Lock()
	wh := s.config.Webhook
	s.mu.Unlock()
	if wh.Enabled {
		msg := fmt.Sprintf("%s: %s %s [ %s ]", m.DisplayName(), result.Status.String(), getResultArrow(prev, result), result.Value)
		go func() {
			err := webhook.Send(ctx, wh.URL, msg)
			if err != nil {
				log.Printf("pushToWebhook:%s:  %v", wh.URL, err)
			}
		}()
	}
}

// handleHealthCheck responds with HTTP 200 OK for health check probes.
// Used for liveness/readiness checks.
func (s *Server) handleHealthCheck(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// handleTestWebhook handles a POST request to test webhook integration.
// Expects a JSON body with a "text" field. Responds with HTTP 200 OK.
func (s *Server) handleTestWebhook(_ context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var hook webhook.Payload
		ok := s.unmarshallBody(w, r, &hook)
		if !ok {
			return
		}
		log.Printf("handleTestWebhook: %v", hook)
		if s.mapper.Impls.Webhook != nil {
			s.mapper.Impls.Webhook(hook.Text)
		}
		http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
	}
}

//go:embed static/*
var staticFS embed.FS

// handleStatic serves static files such as favicon or other assets under /static/.
// Uses Go's embed.FS for serving files. Returns 404 for missing files.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, staticFS, r.URL.Path)
}

//go:embed templates/*
var templateFS embed.FS

// handleTemplate returns an HTTP handler function that renders the main dashboard or configuration page.
// Uses the embedded index.html and config.html templates and injects configuration values.
func (s *Server) handleTemplate() func(w http.ResponseWriter, r *http.Request) {
	// Load templates
	templ, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Printf("handleTemplate (Parse Error): %v", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Internal Server Error: template parse issue, check logs", http.StatusInternalServerError)
		}
	}
	log.Printf("handleTemplate: templates loaded")

	// Handler
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		// select the template based on the URL path
		page := "404.html"
		activeTemplate := "dashboard"
		switch r.URL.Path {
		case "/", "index.html":
			page = "index.html"
			activeTemplate = "dashboard"
		case "/config":
			page = "config.html"
			activeTemplate = "config"
		case "/mobile":
			page = "mobile.html"
			activeTemplate = "mobile"
		default:
			page = r.URL.Path[1:]
		}

		items := s.joinConfigToResults(s.monitor.Results())
		systemItems := make([]UIResult, 0, 2)
		diskItems := make([]UIResult, 0, len(items))
		healthItems := make([]UIResult, 0, len(items))
		mobileItems := make([]UIResult, 0, len(items))
		pingItems := make([]UIResult, 0, len(items))
		for _, item := range items {
			switch item.Group {
			case system.Group:
				systemItems = append(systemItems, item)
			case disk.Group:
				diskItems = append(diskItems, item)
			case healthcheck.Group:
				// Only non-ping healthchecks
				if strings.HasPrefix(item.DisplayName, "Ping: ") {
					pingItems = append(pingItems, item)
				} else {
					healthItems = append(healthItems, item)
				}
			}
			mobileItems = append(mobileItems, item)
		}

		// sort mobileItems by status
		slices.SortFunc(mobileItems, func(a, b UIResult) int {
			return int(a.Status - b.Status)
		})
		// sort systemItems by display Name
		slices.SortFunc(systemItems, displayNameSorter)
		// sort diskItems by display Name
		slices.SortFunc(diskItems, displayNameSorter)
		// sort healthItems by display Name
		slices.SortFunc(healthItems, displayNameSorter)
		// sort pingItems by display Name
		slices.SortFunc(pingItems, displayNameSorter)

		// Set even row for styling in mobile view
		for i, item := range mobileItems {
			item.EvenRow = i%2 == 0
			mobileItems[i] = item
		}

		// Template data
		data := map[string]any{
			"title":               s.config.Monitoring.System.Title,
			"dashboard_title":     s.config.Monitoring.System.Title,
			"refresh_interval":    s.config.Monitoring.Interval,
			"default_disk_icon":   disk.Icon,        // from monitors/disk.icon
			"default_health_icon": healthcheck.Icon, // from monitors/healthcheck.icon
			"SystemItems":         systemItems,
			"DiskItems":           diskItems,
			"HealthItems":         healthItems,
			"PingItems":           pingItems,
			"MobileItems":         mobileItems,
			"ActivePage":          activeTemplate,
			"UpdateAt":            time.Now().Format("2006-01-02 15:04:05Z"),
			"Config":              s.config,
			"IconChoices":         defaultIconList(), // health
		}

		// Execute the template
		t := templ.Lookup(page)
		if t == nil {
			log.Printf("handleTemplate %v: template not found", page)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		err = templ.ExecuteTemplate(w, page, data)
		if err != nil {
			log.Printf("handleTemplate %v: %v", r.URL, err)
			http.Error(w, "Template error - check logs", http.StatusInternalServerError)
			return
		}
	}
}

func displayNameSorter(a, b UIResult) int {
	return strings.Compare(a.DisplayName, b.DisplayName)
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
// Route: GET /api/items
func (s *Server) handleGetItems(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := s.joinConfigToResults(s.monitor.Results())
	s.writeJson(w, results)
}

func (s *Server) joinConfigToResults(items map[string]monitors.Result) map[string]UIResult {
	results := make(map[string]UIResult)
	for id, item := range items {
		results[id] = newUIResult(id, item, s.config)
	}
	return results
}

// handleGetItem responds with a JSON object for a specific monitored item, identified by its group and ID.
// Returns 404 if the item is not found.
// Route: GET /api/items/{group}/{id}
func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := r.PathValue("id")
	group := r.PathValue("group")
	items := s.monitor.Results()
	name := group + "_" + id
	item, ok := items[name]
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusBadRequest)
		log.Printf("handleGetItem: item not found: %s", id)
		return
	}
	s.writeJson(w, newUIResult(name, item, s.config))
}

// handleGetConfig responds with the current server configuration as JSON.
// Route: GET /api/config
func (s *Server) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

// handleUpdateSystemConfig processes a request to update the system configuration.
// Expects a JSON body with the new configuration.
// Route: POST /api/config/system
func (s *Server) handleUpdateSystemConfig(ctx context.Context) http.HandlerFunc {
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

		s.monitor.Add(ctx, cpu)

		// Apply mem config
		mem, err := s.mapper.NewMem(ctx, cfg.Memory)
		if err != nil {
			log.Printf("handleUpdateSystemConfig (mem): %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		s.monitor.Add(ctx, mem)

		s.saveConfig(w)
	}
}

// saveConfig persists the current configuration to disk and responds with HTTP 200 OK on success.
// If saving fails, responds with HTTP 500.
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
// Expects a JSON body with an "Interval" field (seconds).
// Route: POST /api/config/interval
func (s *Server) handleIntervalUpdate(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		var cfg struct {
			Interval int
		}
		ok := s.unmarshallBody(w, r, &cfg)
		if !ok {
			return
		}
		s.monitor.SetPeriod(ctx, time.Duration(cfg.Interval)*time.Second, 0)
		s.saveConfig(w)
	}
}

// handleAddDiskMonitor processes a request to add a new disk monitor.
// Expects a JSON body describing the disk monitor to add.
// Route: POST /api/config/disk/{id}
func (s *Server) handleAddDiskMonitor(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		id := r.PathValue("id")
		var cfg config.DiskConfig
		ok := s.unmarshallBody(w, r, &cfg)
		if !ok {
			return
		}

		d, err := s.mapper.NewDisk(ctx, id, cfg)
		if err != nil {
			log.Printf("handleAddDiskMonitor %s: %v", r.URL.String(), err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.monitor.Add(ctx, d)

		s.saveConfig(w)
	}
}

// handleAddHealthCheck processes a request to add a new health check monitor.
// Expects a JSON body describing the health check to add.
// Route: POST /api/config/health/{id}
func (s *Server) handleAddHealthCheck(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		id := r.PathValue("id")
		var cfg config.HealthcheckConfig
		ok := s.unmarshallBody(w, r, &cfg)
		if !ok {
			return
		}

		d, err := s.mapper.NewHealthcheck(ctx, id, cfg)
		if err != nil {
			log.Printf("handleAddHealthCheck %s: %v", r.URL.String(), err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.monitor.Add(ctx, d)

		s.saveConfig(w)
	}
}

// handleUpdateWebhook processes a request to update the webhook configuration.
// Expects a JSON body with the new webhook settings.
// Route: POST /api/config/webhook
func (s *Server) handleUpdateWebhook(_ context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		var cfg config.WebhookConfig
		ok := s.unmarshallBody(w, r, &cfg)
		if !ok {
			return
		}

		// Save webhook config
		s.config.Webhook = cfg

		s.saveConfig(w)
	}

}

// handleDeleteMonitor processes a request to delete a monitor by type and ID.
// Route: DELETE /api/config/{type}/{id}
func (s *Server) handleDeleteMonitor(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		t := r.PathValue("type")
		id := r.PathValue("id")
		var m monitors.Monitor
		switch t {
		case "disk":
			m, _ = s.mapper.NewDisk(ctx, id, config.DiskConfig{})
		case "health":
			m, _ = s.mapper.NewHealthcheck(ctx, id, config.HealthcheckConfig{})
		case "ping":
			// Use a dummy config with valid amberThreshold so Name() is correct
			m, _ = s.mapper.NewPing(ctx, id, config.PingConfig{AmberThreshold: 1})
		default:
			log.Printf("handleDeleteMonitor invalid type: %s", r.URL.String())
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		err := s.monitor.Remove(m)
		if err != nil {
			log.Printf("handleDeleteMonitor %s: %v", r.URL.String(), err)
			if errors.As(err, &monitors.ErrNotFound{}) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Ensure config is updated after ping monitor deletion
		s.saveConfig(w)
	}
}

// handleAddPingMonitor processes a request to add a new ping monitor.
// Expects a JSON body describing the ping monitor to add.
// Route: POST /api/config/ping/{id}
func (s *Server) handleAddPingMonitor(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		id := r.PathValue("id")
		var cfg config.PingConfig
		ok := s.unmarshallBody(w, r, &cfg)
		if !ok {
			return
		}

		p, err := s.mapper.NewPing(ctx, id, cfg)
		if err != nil {
			log.Printf("handleAddPingMonitor %s: %v", r.URL.String(), err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.monitor.Add(ctx, p)

		s.saveConfig(w)
	}
}

// SetConfig applies the provided monitoring configuration to the server and monitoring service.
// This includes system, disk, and healthcheck monitors, as well as the monitoring interval.
// Returns an error if any monitor fails to initialize or add.
func (s *Server) SetConfig(ctx context.Context, cfg config.MonitoringConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cpu, err := s.mapper.NewCpu(ctx, cfg.System.CPU)
	if err != nil {
		return err
	}
	s.monitor.Add(ctx, cpu)
	log.Printf("Set CPU %v", cpu)

	mem, err := s.mapper.NewMem(ctx, cfg.System.Memory)
	if err != nil {
		return err
	}
	s.monitor.Add(ctx, mem)
	log.Printf("Set MEM %v", cpu)

	for name, i := range cfg.Disk {
		newDisk, err := s.mapper.NewDisk(ctx, name, i)
		if err != nil {
			return err
		}
		s.monitor.Add(ctx, newDisk)
		log.Printf("Added Disk %v", newDisk)
	}

	for name, i := range cfg.Healthcheck {
		newHealth, err := s.mapper.NewHealthcheck(ctx, name, i)
		if err != nil {
			return err
		}
		s.monitor.Add(ctx, newHealth)
		log.Printf("Added Healthcheck %v", newHealth)
	}

	for name, i := range cfg.Ping {
		if i.AmberThreshold <= 0 {
			return fmt.Errorf("Ping monitor '%s' missing required amberThreshold", name)
		}
		newPing, err := s.mapper.NewPing(ctx, name, i)
		if err != nil {
			return err
		}
		s.monitor.Add(ctx, newPing)
		log.Printf("Added Ping %v", newPing)
	}

	// Set the monitoring interval to trigger the initial checks.
	s.monitor.SetPeriod(ctx, time.Duration(cfg.Interval)*time.Second, 0)

	log.Printf("Configuration applied")

	return nil

}
