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

	// Load configuration and initialize cache
	tasks, err := LoadConfig(opt.Config)
	if err != nil {
		os.Exit(1)
	}
	cache, err := NewCache()
	if err != nil {
		os.Exit(1)
	}

	// Create context and wait group for goroutines
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start tasks in separate goroutines
	for _, task := range *tasks {
		wg.Add(1)
		go func(task *Task) {
			defer wg.Done()
			task.Start(ctx, cache)
		}(task)
		time.Sleep(5 * time.Second)
	}

	// Handle termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancel()

	// Wait for all tasks to finish
	wg.Wait()
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
