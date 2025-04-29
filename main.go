/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"log/slog"
	"net/http" // Added for web server
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jessevdk/go-flags"
)

type options struct {
	Config           string `short:"c" long:"conf" description:"Config file" default:"at-rss.conf"`
	WebListenAddress string `long:"web-listen" description:"Listen address for the web UI/API (e.g., ':8080'). If empty, web server is disabled." default:""`
	WebUIDir         string `long:"web-ui-dir" description:"Directory containing the web UI static files (index.html, etc.)" default:"webui/dist"`
	Token            string `long:"token" description:"Token for API authentication. If empty, no authentication is required." default:""`
	FetchInterval    int    `long:"default-fetch-interval" description:"Default fetch interval in minutes (overrides config default)" default:"0"`
}

var opt options
var parser = flags.NewParser(&opt, flags.Default)
var webServer *http.Server // Global variable to hold the server instance

func main() {
	if _, err := parser.Parse(); err != nil {
		handleFlagsError(err)
	}

	// Init watcher for reload configure files
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to create file watcher", "error", err)
		return
	}
	defer watcher.Close()
	err = watcher.Add(opt.Config)
	if err != nil {
		slog.Error("Can't watch configure file", "error", err)
		return
	}

	cache, err := NewCache()
	if err != nil {
		slog.Error("Failed to initialize cache", "error", err)
		return
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	atRSS := func(ctx context.Context) error {
		tasks, err := LoadConfig(opt.Config, opt.FetchInterval)
		if err != nil {
			slog.Error("Failed to load config", "error", err)
			return err
		}
		if len(tasks) == 0 {
			slog.Warn("No task is running")
			return nil
		}
		for _, task := range tasks {
			wg.Add(1)
			go func(task *Task) {
				defer wg.Done()
				task.Start(ctx, cache)
			}(task)
			time.Sleep(5 * time.Second)
		}
		return nil
	}
	if err := atRSS(ctx); err != nil {
		return // Exit if initial config load fails
	}

	// --- Start Web Server (if configured) ---
	var errWeb error
	if opt.WebListenAddress != "" {
		// Pass the actual config path and token being used
		webServer, errWeb = StartWebServer(opt.WebListenAddress, opt.WebUIDir, opt.Config, opt.Token)
		if errWeb != nil {
			slog.Error("Failed to start web server", "error", errWeb)
			// Decide if this is fatal. For now, let's log and continue without web UI.
			// return
		}
	} else {
		slog.Info("Web server is disabled (web-listen address not provided).")
	}
	// --- End Web Server Start ---

	var debounceTimer *time.Timer
	debounceDuration := 5 * time.Second
	for {
		select {
		case <-stop:
			slog.Info("Shutting down...")

			// --- Graceful Shutdown for Web Server ---
			if webServer != nil {
				slog.Info("Stopping web server...")
				// Create a context with timeout for shutdown
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second) // Use imported time
				defer shutdownCancel()

				if err := webServer.Shutdown(shutdownCtx); err != nil { // Use imported context
					slog.Error("Web server shutdown failed", "error", err)
				} else {
					slog.Info("Web server stopped.")
				}
			}
			// --- End Graceful Shutdown ---

			cancel()  // Signal tasks to stop
			wg.Wait() // Wait for all tasks to finish
			slog.Info("All tasks stopped. Exiting.")
			return // Exit main function
		case event, ok := <-watcher.Events:
			if !ok {
				slog.Error("Configure file watching error", "error", err)
				return
			}
			if event.Has(fsnotify.Write) {
				if debounceTimer == nil {
					debounceTimer = time.AfterFunc(debounceDuration, func() {
						slog.Info("Reloading configure file...")
						slog.Info("Stopping tasks for reload...")
						cancel()  // Signal current tasks to stop
						wg.Wait() // Wait for tasks to finish before reloading
						slog.Info("Tasks stopped.")
						ctx, cancel = context.WithCancel(context.Background())
						if err := atRSS(ctx); err != nil {
							// If reload fails, we should probably stop the application
							// as the state might be inconsistent.
							slog.Error("Failed to reload config and restart tasks", "error", err)
							// Consider stopping the program here:
							// stop <- syscall.SIGTERM // Send signal to trigger shutdown sequence
							return
						}
						debounceTimer = nil
						slog.Info("Configure file reloaded.")
					})
				} else {
					debounceTimer.Reset(debounceDuration)
				}
			}
		}
	}
}

// handleFlagsError processes errors from flag parsing
func handleFlagsError(err error) {
	if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
		os.Exit(0)
	} else {
		slog.Error("Flag parsing error", "error", err)
		os.Exit(1)
	}
}
