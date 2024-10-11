/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jessevdk/go-flags"
)

type options struct {
	Config string `short:"c" long:"conf" description:"Config file" default:"/etc/at-rss.conf"`
}

var opt options
var parser = flags.NewParser(&opt, flags.Default)

func main() {
	// Parse command line arguments
	if _, err := parser.Parse(); err != nil {
		handleFlagsError(err)
	}

	// Init watcher for reload configure files
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		os.Exit(1)
	}
	defer watcher.Close()
	err = watcher.Add(opt.Config)
	if err != nil {
		slog.Error("Can't watch configure file.")
		os.Exit(1)
	}

	// Handle termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init task manager
	at_rss := func(ctx context.Context) {
		// Init cache for parsing torrent files
		cache, err := NewCache(ctx)
		if err != nil {
			os.Exit(1)
		}
		defer cache.Flush()

		tasks, err := LoadConfig(opt.Config)
		if err != nil {
			os.Exit(1)
		}
		// Start tasks in separate goroutines
		if len(*tasks) == 0 {
			slog.Warn("No task is running.")
		}
		for _, task := range *tasks {
			wg.Add(1)
			go func(task *Task) {
				defer wg.Done()
				task.Start(ctx, cache)
			}(task)
			time.Sleep(5 * time.Second) // Optional delay between starting tasks
		}
	}

	at_rss(ctx)
	var reloadTimer *time.Timer

	for {
		select {
		case <-stop:
			cancel()
			wg.Wait()
			return
		case event, ok := <-watcher.Events:
			if !ok {
				slog.Error("Configure file watching error", "error:", err)
				return
			}
			// When configure file changes, reset reload timer
			if event.Op&fsnotify.Write == fsnotify.Write {
				if reloadTimer != nil {
					reloadTimer.Stop()
				}
				reloadTimer = time.AfterFunc(5*time.Second, func() {
					cancel()
					wg.Wait()

					ctx, cancel = context.WithCancel(context.Background())
					at_rss(ctx)
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				slog.Error("Configure file watching error", "error:", err)
				return
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
