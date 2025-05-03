package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// --- Downloaders Management ---

type downloaderInfo struct {
	dc        ParsedDownloaderConfig
	TaskNames []string
}

// DownloaderGroup holds merged downloader information grouped by RPC URL.
// The contents are initialized once and become immutable.
type DownloaderGroup struct {
	ctx context.Context
	m   map[string]downloaderInfo // map of RPC URL to downloader info
}

var (
	downloaderGroup *DownloaderGroup
)

// getUniqueDownloaders builds the global downloader map with information from tasks
func getUniqueDownloaders(ctx context.Context, tasks []*Task) *DownloaderGroup {
	downloaderGroup = &DownloaderGroup{
		ctx: ctx,
		m:   make(map[string]downloaderInfo),
	}

	for _, task := range tasks {
		for _, dlConfig := range task.Downloaders {
			info, exists := downloaderGroup.m[dlConfig.RpcUrl]
			if !exists {
				info = downloaderInfo{
					dc:        dlConfig,
					TaskNames: []string{task.Name},
				}
			} else {
				// Append task name if not already present
				found := slices.Contains(info.TaskNames, task.Name)
				if !found {
					info.TaskNames = append(info.TaskNames, task.Name)
				}
			}
			downloaderGroup.m[dlConfig.RpcUrl] = info
		}
	}
	return downloaderGroup
}

// --- Download Status Management ---

// DownloadStatusPublisher manages download status subscriptions
type DownloadStatusPublisher struct {
	subscribers   map[chan []DownloadStatus]struct{}
	lastStatus    []DownloadStatus
	rpcClients    map[string]RpcClient
	rpcUrlCounter map[string]int // tracks active subscriptions per RPC URL
	active        bool
	stopChan      chan struct{}
	lastActive    time.Time
	sync.RWMutex
}

func NewDownloadStatusPublisher() *DownloadStatusPublisher {
	return &DownloadStatusPublisher{
		subscribers:   make(map[chan []DownloadStatus]struct{}),
		rpcClients:    make(map[string]RpcClient),
		rpcUrlCounter: make(map[string]int),
		stopChan:      make(chan struct{}),
	}
}

func (p *DownloadStatusPublisher) Subscribe(rpcUrl string) chan []DownloadStatus {
	p.Lock()
	defer p.Unlock()

	ch := make(chan []DownloadStatus, 1)
	p.subscribers[ch] = struct{}{}
	p.lastActive = time.Now()

	// Update counter for RPC URLs
	if rpcUrl != "" {
		p.rpcUrlCounter[rpcUrl]++
	} else {
		// When rpcUrl is empty, increment all downloaders' counters
		for url := range downloaderGroup.m {
			p.rpcUrlCounter[url]++
		}
	}

	// Start publisher if not active
	if !p.active {
		p.active = true
		go p.run()
	}

	// Send initial status if available
	if len(p.lastStatus) > 0 {
		select {
		case ch <- p.lastStatus:
		default:
			// Skip if initial status is not ready
		}
	}
	return ch
}

func (p *DownloadStatusPublisher) Unsubscribe(ch chan []DownloadStatus, rpcUrl string) {
	p.Lock()
	defer p.Unlock()

	delete(p.subscribers, ch)
	close(ch)

	// Update counter for RPC URLs
	if rpcUrl != "" {
		if count, exists := p.rpcUrlCounter[rpcUrl]; exists {
			if count <= 1 {
				delete(p.rpcUrlCounter, rpcUrl)
			} else {
				p.rpcUrlCounter[rpcUrl]--
			}
		}
	} else {
		// When rpcUrl is empty, decrement all downloaders' counters
		for url := range downloaderGroup.m {
			if count, exists := p.rpcUrlCounter[url]; exists {
				if count <= 1 {
					delete(p.rpcUrlCounter, url)
				} else {
					p.rpcUrlCounter[url]--
				}
			}
		}
	}
}

func (p *DownloadStatusPublisher) Update(status []DownloadStatus) {
	p.Lock()
	defer p.Unlock()

	p.lastStatus = status
	p.lastActive = time.Now()
	for ch := range p.subscribers {
		select {
		case ch <- status:
		default:
			// Skip if subscriber is not ready
		}
	}
}

func (p *DownloadStatusPublisher) run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	idleTimeout := 30 * time.Second

	for {
		select {
		case <-ticker.C:
			p.RLock()
			subscriberCount := len(p.subscribers)
			lastActive := p.lastActive
			p.RUnlock()

			if subscriberCount == 0 && time.Since(lastActive) > idleTimeout {
				p.Lock()
				p.active = false
				// Close all RPC clients
				for _, client := range p.rpcClients {
					client.CloseRpc()
				}
				p.rpcClients = make(map[string]RpcClient)
				p.rpcUrlCounter = make(map[string]int)
				p.Unlock()
				return
			}

			// Process only RPC URLs with active subscriptions
			p.RLock()
			activeRpcUrls := make([]string, 0, len(p.rpcUrlCounter))
			for rpcUrl := range p.rpcUrlCounter {
				activeRpcUrls = append(activeRpcUrls, rpcUrl)
			}
			p.RUnlock()

			// If no active RPC URLs, skip processing
			if len(activeRpcUrls) == 0 {
				continue
			}

			// Prepare clients first (serial)
			clients := make(map[string]RpcClient, len(activeRpcUrls))
			p.Lock()
			for _, rpcUrl := range activeRpcUrls {
				info, exists := downloaderGroup.m[rpcUrl]
				if !exists {
					continue
				}

				// Create client if not exists
				if _, exists := p.rpcClients[rpcUrl]; !exists {
					client, err := createRpcClientForConfig(downloaderGroup.ctx, info.dc)
					if err != nil {
						slog.Error("Failed to create RPC client", "rpcUrl", rpcUrl, "error", err)
						continue
					}
					p.rpcClients[rpcUrl] = client
				}
				clients[rpcUrl] = p.rpcClients[rpcUrl]
			}
			p.Unlock()

			// Process downloads in parallel
			for rpcUrl, client := range clients {
				go func(url string, c RpcClient) {
					status, err := c.GetActiveDownloads()
					if err != nil {
						slog.Error("Failed to get active downloads", "rpcUrl", url, "error", err)
						return
					}

					if len(status) > 0 {
						p.Update(status)
					}
				}(rpcUrl, client)
			}

		case <-p.stopChan:
			return
		case <-downloaderGroup.ctx.Done():
			return
		}
	}
}

func (p *DownloadStatusPublisher) Stop() {
	p.Lock()
	defer p.Unlock()

	if p.active {
		close(p.stopChan)
		p.active = false
	}
}

var (
	statusPublisher *DownloadStatusPublisher
	publisherMutex  sync.Mutex
)

func getStatusPublisher() *DownloadStatusPublisher {
	publisherMutex.Lock()
	defer publisherMutex.Unlock()

	if statusPublisher == nil {
		statusPublisher = NewDownloadStatusPublisher()
	}
	return statusPublisher
}

// --- Helpers ---

func parseRequest[T any](w http.ResponseWriter, r *http.Request, target T) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendError(w, "Failed to read request body", http.StatusBadRequest, "error", err)
		return false
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, target); err != nil {
		sendError(w, fmt.Sprintf("Invalid JSON format: %s", err), http.StatusBadRequest)
		return false
	}
	return true
}

func validateTaskRequest(w http.ResponseWriter, name string, config TaskConfig) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		sendError(w, "Task name cannot be empty", http.StatusBadRequest)
		return false
	}

	if len(config.Downloaders) == 0 {
		sendError(w, "Task must have at least one downloader", http.StatusBadRequest)
		return false
	}
	if len(config.Feeds) == 0 {
		sendError(w, "Task must have at least one feed", http.StatusBadRequest)
		return false
	}
	return true
}

func sendError(w http.ResponseWriter, message string, code int, args ...any) {
	slog.Error("API: "+message, args...)
	http.Error(w, message, code)
}

func sendJSONResponse(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("API: Failed to encode response to JSON", "error", err)
	}
}

// --- HTTP Handler Factories ---

// handleDownloaders creates a handler function for the /api/downloaders endpoint
func handleDownloaders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		response := make(map[string][]string)
		for rpcUrl, info := range downloaderGroup.m {
			response[rpcUrl] = info.TaskNames
		}

		sendJSONResponse(w, http.StatusOK, response)
	}
}

// handleDownloads creates a handler function for the /api/downloads endpoint with SSE support
func handleDownloads() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Get requested RPC URL from header
		rpcUrl := r.Header.Get("X-Rpc-Url")

		// Validate RPC URL if specified
		if rpcUrl != "" {
			if _, exists := downloaderGroup.m[rpcUrl]; !exists {
				slog.Error("Invalid RPC URL requested", "rpcUrl", rpcUrl)
				sendError(w, "Invalid RPC URL specified", http.StatusBadRequest)
				return
			}
		}

		// Subscribe to status updates
		publisher := getStatusPublisher()
		statusCh := publisher.Subscribe(rpcUrl)
		defer publisher.Unsubscribe(statusCh, rpcUrl)

		// Create a channel to detect client disconnection using context
		clientGone := r.Context().Done()

		for {
			select {
			case status := <-statusCh:
				// Filter status by RPC URL if specified
				filteredStatus := status
				if rpcUrl != "" {
					filteredStatus = []DownloadStatus{}
					for _, s := range status {
						if s.RpcUrl == rpcUrl {
							filteredStatus = append(filteredStatus, s)
						}
					}
				}

				if len(filteredStatus) > 0 {
					data, err := json.Marshal(filteredStatus)
					if err != nil {
						slog.Error("Failed to marshal download status", "error", err)
						continue
					}
					if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
						slog.Error("Failed to write SSE data", "error", err)
						return
					}
					w.(http.Flusher).Flush()
				}
			case <-clientGone:
				// Client disconnected
				slog.Debug("Client disconnected from SSE stream")
				return
			case <-downloaderGroup.ctx.Done():
				// Config file reloading...
				slog.Debug("Config file reloading...Stop SSE stream")
				return
			}
		}
	}
}

// handleTasks creates a handler function for the /api/tasks endpoint.
func handleTasks(cfgPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getAllTasks(w, r, cfgPath)
		case http.MethodPost:
			createTask(w, r, cfgPath)
		default:
			sendError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleSingleTask creates a handler function for /api/tasks/{taskName}.
func handleSingleTask(cfgPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract task name robustly, handling potential trailing slashes
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 3 || pathParts[2] == "" { // Expecting /api/tasks/{taskName}
			sendError(w, "Task name missing or invalid in URL path", http.StatusBadRequest)
			return
		}
		taskName := pathParts[2]

		switch r.Method {
		case http.MethodGet:
			getTaskByName(w, r, cfgPath, taskName)
		case http.MethodPut:
			updateTask(w, r, cfgPath, taskName)
		case http.MethodDelete:
			deleteTask(w, r, cfgPath, taskName)
		default:
			sendError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// getAllTasks retrieves all task configurations.
func getAllTasks(w http.ResponseWriter, r *http.Request, cfgPath string) {
	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		sendError(w, "Failed to load configuration", http.StatusInternalServerError, "error", err, "path", cfgPath)
		return
	}

	sendJSONResponse(w, http.StatusOK, tasks)
}

// createTask creates a new task configuration.
func createTask(w http.ResponseWriter, r *http.Request, cfgPath string) {
	// Parse request
	var newTaskReq struct {
		Name   string     `json:"name"`
		Config TaskConfig `json:"config"`
	}
	if !parseRequest(w, r, &newTaskReq) {
		return
	}

	if !validateTaskRequest(w, newTaskReq.Name, newTaskReq.Config) {
		return
	}

	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		sendError(w, "Failed to load configuration", http.StatusInternalServerError, "error", err, "path", cfgPath)
		return
	}
	if _, exists := tasks[newTaskReq.Name]; exists {
		sendError(w, fmt.Sprintf("Task with name '%s' already exists", newTaskReq.Name), http.StatusConflict)
		return
	}
	tasks[newTaskReq.Name] = newTaskReq.Config
	if err := SaveYAMLConfig(cfgPath, tasks); err != nil {
		sendError(w, "Failed to save configuration", http.StatusInternalServerError, "error", err, "path", cfgPath)
		return
	}

	sendJSONResponse(w, http.StatusCreated, newTaskReq.Config)
}

// getTaskByName retrieves a specific task configuration.
func getTaskByName(w http.ResponseWriter, r *http.Request, cfgPath string, taskName string) {
	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		sendError(w, "Failed to load configuration", http.StatusInternalServerError, "error", err, "path", cfgPath)
		return
	}

	task, exists := tasks[taskName]
	if !exists {
		http.Error(w, fmt.Sprintf("Task '%s' not found", taskName), http.StatusNotFound)
		return
	}

	sendJSONResponse(w, http.StatusOK, task)
}

// updateTask updates an existing task configuration.
func updateTask(w http.ResponseWriter, r *http.Request, cfgPath string, taskName string) {
	var updatedConfig TaskConfig
	if !parseRequest(w, r, &updatedConfig) {
		return
	}

	if !validateTaskRequest(w, taskName, updatedConfig) {
		return
	}

	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		sendError(w, "Failed to load configuration", http.StatusInternalServerError, "error", err, "path", cfgPath)
		return
	}
	if _, exists := tasks[taskName]; !exists {
		sendError(w, fmt.Sprintf("Task '%s' not found", taskName), http.StatusNotFound)
		return
	}
	tasks[taskName] = updatedConfig
	if err := SaveYAMLConfig(cfgPath, tasks); err != nil {
		sendError(w, "Failed to save configuration", http.StatusInternalServerError, "error", err, "path", cfgPath)
		return
	}

	sendJSONResponse(w, http.StatusOK, updatedConfig)
}

// deleteTask removes a task configuration.
func deleteTask(w http.ResponseWriter, r *http.Request, cfgPath string, taskName string) {
	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		sendError(w, "Failed to load configuration", http.StatusInternalServerError, "error", err, "path", cfgPath)
		return
	}

	if _, exists := tasks[taskName]; !exists {
		sendError(w, fmt.Sprintf("Task '%s' not found", taskName), http.StatusNotFound)
		return
	}

	delete(tasks, taskName)

	if err := SaveYAMLConfig(cfgPath, tasks); err != nil {
		sendError(w, "Failed to save configuration", http.StatusInternalServerError, "error", err, "path", cfgPath)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Task '%s' deleted successfully", taskName) // Simple text response for delete
}

// --- Web Server Setup ---

// authMiddleware wraps a handler with token authentication if token is not empty
func authMiddleware(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for static files and when token is empty
		if strings.HasPrefix(r.URL.Path, "/api") && token != "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized: Missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}

			providedToken := strings.TrimPrefix(authHeader, "Bearer ")
			if providedToken != token {
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

// StartWebServer initializes and starts the HTTP server for the API and static UI files.
// It accepts the listen address, UI directory path, config file path and optional token.
// Returns the http.Server instance for graceful shutdown and any error during setup.
func StartWebServer(addr string, webUiDir string, cfgPath string, token string) (*http.Server, error) {
	mux := http.NewServeMux()

	// --- API Routes ---
	// Use closures to pass the config path to the handler factories
	// Wrap API handlers with auth middleware if token is provided
	mux.HandleFunc("/api/tasks", authMiddleware(token, handleTasks(cfgPath)))
	mux.HandleFunc("/api/tasks/", authMiddleware(token, handleSingleTask(cfgPath))) // Trailing slash handles /api/tasks/{name}
	mux.HandleFunc("/api/downloads", authMiddleware(token, handleDownloads()))
	mux.HandleFunc("/api/downloaders", authMiddleware(token, handleDownloaders()))

	// --- Static File Serving ---
	if webUiDir != "" {
		// Check if the directory exists
		if _, err := os.Stat(webUiDir); os.IsNotExist(err) {
			slog.Warn("Web UI directory does not exist. Static files will not be served.", "directory", webUiDir)
			// Optionally create it:
			// slog.Info("Creating Web UI directory.", "directory", webUiDir)
			// if err := os.MkdirAll(webUiDir, 0755); err != nil {
			// 	slog.Error("Failed to create Web UI directory", "directory", webUiDir, "error", err)
			// 	// Decide whether to proceed without UI or return error
			// }
		} else {
			// Create a file server handler
			fs := http.FileServer(http.Dir(webUiDir))

			// Handle requests for static files and SPA routing
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				// Prevent directory listing by redirecting or returning 404 for "/" if index.html doesn't exist
				if r.URL.Path == "/" {
					indexPath := filepath.Join(webUiDir, "index.html")
					if _, err := os.Stat(indexPath); os.IsNotExist(err) {
						http.NotFound(w, r) // Or serve a custom "UI not found" page
						return
					}
					// Serve index.html for the root
					http.ServeFile(w, r, indexPath)
					return
				}

				// Construct the potential file path
				filePath := filepath.Join(webUiDir, filepath.Clean(r.URL.Path))

				// Check if the file exists
				if _, err := os.Stat(filePath); err != nil {
					if os.IsNotExist(err) {
						// File doesn't exist, assume it's an SPA route
						// Serve index.html to let the frontend handle routing
						indexPath := filepath.Join(webUiDir, "index.html")
						if _, indexErr := os.Stat(indexPath); indexErr == nil {
							http.ServeFile(w, r, indexPath)
						} else {
							// index.html not found either
							http.NotFound(w, r)
						}
						return
					}
					// Other error (e.g., permissions)
					slog.Error("Error checking static file", "path", filePath, "error", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				// File exists, serve it using the file server
				fs.ServeHTTP(w, r)
			})
			slog.Info("Serving static files for Web UI", "directory", webUiDir)
		}
	} else {
		slog.Warn("Web UI directory not specified. Only API endpoints will be available.")
		// Add a root handler for API-only mode if desired
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				fmt.Fprintln(w, "at-rss API is running. No Web UI configured.")
			} else {
				http.NotFound(w, r)
			}
		})
	}

	// Create the server instance
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		// Add timeouts for production hardening
		// ReadTimeout:  5 * time.Second,
		// WriteTimeout: 10 * time.Second,
		// IdleTimeout:  120 * time.Second,
	}

	// Start the server in a separate goroutine so it doesn't block
	go func() {
		slog.Info("Starting web server", "address", addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			// Log error, consider signaling main thread for critical failure
			slog.Error("Web server ListenAndServe failed", "error", err)
		}
	}()

	return server, nil // Return the server instance for graceful shutdown management
}
