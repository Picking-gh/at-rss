package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// --- HTTP Handler Factories ---

// handleTasks creates a handler function for the /api/tasks endpoint.
func handleTasks(cfgPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getAllTasks(w, r, cfgPath)
		case http.MethodPost:
			createTask(w, r, cfgPath)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleSingleTask creates a handler function for /api/tasks/{taskName}.
func handleSingleTask(cfgPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract task name robustly, handling potential trailing slashes
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 3 || pathParts[2] == "" { // Expecting /api/tasks/{taskName}
			http.Error(w, "Task name missing or invalid in URL path", http.StatusBadRequest)
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
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// --- Specific Request Handlers ---

// getAllTasks retrieves all task configurations.
func getAllTasks(w http.ResponseWriter, r *http.Request, cfgPath string) {
	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		slog.Error("API: Failed to load config data", "error", err, "path", cfgPath)
		http.Error(w, "Failed to load configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		slog.Error("API: Failed to encode tasks to JSON", "error", err)
		// Avoid writing header again if already started
	}
}

// createTask creates a new task configuration.
func createTask(w http.ResponseWriter, r *http.Request, cfgPath string) {
	var newTaskReq struct {
		Name   string     `json:"name"`
		Config TaskConfig `json:"config"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("API: Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &newTaskReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON format: %s", err), http.StatusBadRequest)
		return
	}

	// Trim whitespace from name
	newTaskReq.Name = strings.TrimSpace(newTaskReq.Name)
	if newTaskReq.Name == "" {
		http.Error(w, "Task name cannot be empty", http.StatusBadRequest)
		return
	}

	// Basic validation (mirroring frontend, but important for API robustness)
	if len(newTaskReq.Config.Downloaders) == 0 {
		http.Error(w, "Task must have at least one downloader", http.StatusBadRequest)
		return
	}
	if len(newTaskReq.Config.Feed.URLs) == 0 {
		http.Error(w, "Task must have at least one feed URL", http.StatusBadRequest)
		return
	}
	// Add more validation based on config.go logic if needed (e.g., interval > 0)

	// Load current config to check for conflicts and merge
	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		slog.Error("API: Failed to load config data before creating task", "error", err, "path", cfgPath)
		http.Error(w, "Failed to load configuration", http.StatusInternalServerError)
		return
	}

	if _, exists := tasks[newTaskReq.Name]; exists {
		http.Error(w, fmt.Sprintf("Task with name '%s' already exists", newTaskReq.Name), http.StatusConflict)
		return
	}

	// Add the new task
	tasks[newTaskReq.Name] = newTaskReq.Config

	// Save the updated config
	if err := SaveYAMLConfig(cfgPath, tasks); err != nil {
		slog.Error("API: Failed to save config data after creating task", "error", err, "path", cfgPath)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json") // Return the created task config
	w.WriteHeader(http.StatusCreated)
	// Encode the newly added task config back to the client
	if err := json.NewEncoder(w).Encode(newTaskReq.Config); err != nil {
		slog.Error("API: Failed to encode created task to JSON", "error", err, "taskName", newTaskReq.Name)
		// If encoding fails after status created, log it but can't change response
	}
}

// getTaskByName retrieves a specific task configuration.
func getTaskByName(w http.ResponseWriter, r *http.Request, cfgPath string, taskName string) {
	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		slog.Error("API: Failed to load config data", "error", err, "path", cfgPath)
		http.Error(w, "Failed to load configuration", http.StatusInternalServerError)
		return
	}

	task, exists := tasks[taskName]
	if !exists {
		http.Error(w, fmt.Sprintf("Task '%s' not found", taskName), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		slog.Error("API: Failed to encode task to JSON", "error", err, "taskName", taskName)
	}
}

// updateTask updates an existing task configuration.
func updateTask(w http.ResponseWriter, r *http.Request, cfgPath string, taskName string) {
	var updatedConfig TaskConfig

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("API: Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &updatedConfig); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON format: %s", err), http.StatusBadRequest)
		return
	}

	// Basic validation
	if len(updatedConfig.Downloaders) == 0 {
		http.Error(w, "Task must have at least one downloader", http.StatusBadRequest)
		return
	}
	if len(updatedConfig.Feed.URLs) == 0 {
		http.Error(w, "Task must have at least one feed URL", http.StatusBadRequest)
		return
	}
	// Add more validation as needed

	// Load current config to ensure task exists and update it
	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		slog.Error("API: Failed to load config data before updating task", "error", err, "path", cfgPath)
		http.Error(w, "Failed to load configuration", http.StatusInternalServerError)
		return
	}

	if _, exists := tasks[taskName]; !exists {
		http.Error(w, fmt.Sprintf("Task '%s' not found", taskName), http.StatusNotFound)
		return
	}

	// Update the task
	tasks[taskName] = updatedConfig

	// Save the updated config
	if err := SaveYAMLConfig(cfgPath, tasks); err != nil {
		slog.Error("API: Failed to save config data after updating task", "error", err, "path", cfgPath)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json") // Return the updated task config
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedConfig); err != nil {
		slog.Error("API: Failed to encode updated task to JSON", "error", err, "taskName", taskName)
	}
}

// deleteTask removes a task configuration.
func deleteTask(w http.ResponseWriter, r *http.Request, cfgPath string, taskName string) {
	tasks, err := LoadYAMLConfig(cfgPath)
	if err != nil {
		slog.Error("API: Failed to load config data before deleting task", "error", err, "path", cfgPath)
		http.Error(w, "Failed to load configuration", http.StatusInternalServerError)
		return
	}

	if _, exists := tasks[taskName]; !exists {
		http.Error(w, fmt.Sprintf("Task '%s' not found", taskName), http.StatusNotFound)
		return
	}

	// Delete the task
	delete(tasks, taskName)

	// Save the updated config
	if err := SaveYAMLConfig(cfgPath, tasks); err != nil {
		slog.Error("API: Failed to save config data after deleting task", "error", err, "path", cfgPath)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Task '%s' deleted successfully", taskName) // Simple text response for delete
}

// --- Web Server Setup ---

// StartWebServer initializes and starts the HTTP server for the API and static UI files.
// It accepts the listen address, UI directory path, and config file path.
// Returns the http.Server instance for graceful shutdown and any error during setup.
func StartWebServer(addr string, webUiDir string, cfgPath string) (*http.Server, error) {
	mux := http.NewServeMux()

	// --- API Routes ---
	// Use closures to pass the config path to the handler factories
	mux.HandleFunc("/api/tasks", handleTasks(cfgPath))
	mux.HandleFunc("/api/tasks/", handleSingleTask(cfgPath)) // Trailing slash handles /api/tasks/{name}

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
