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
		tasks, err := LoadConfig(opt.Config)
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
		return
	}

	var debounceTimer *time.Timer
	debounceDuration := 5 * time.Second
	for {
		select {
		case <-stop:
			cancel()
			wg.Wait()
			return
		case event, ok := <-watcher.Events:
			if !ok {
				slog.Error("Configure file watching error", "error", err)
				return
			}
			if event.Has(fsnotify.Write) {
				if debounceTimer == nil {
					debounceTimer = time.AfterFunc(debounceDuration, func() {
						slog.Info("Reloading configure file...")
						cancel()
						wg.Wait()
						ctx, cancel = context.WithCancel(context.Background())
						if err := atRSS(ctx); err != nil {
							slog.Error("Failed to reload config", "error", err)
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
