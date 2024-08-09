/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jessevdk/go-flags"
)

type options struct {
	Config string `short:"c" long:"conf" description:"Config file" default:"/etc/aria2c-rss.conf"`
}

var opt options

var parser = flags.NewParser(&opt, flags.Default)

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	tasks, err := LoadConfig(opt.Config)
	if err != nil {
		os.Exit(1)
	}
	cache, err := NewCache()
	if err != nil {
		os.Exit(1)
	}

	var wg sync.WaitGroup

	for _, task := range *tasks {
		wg.Add(1)
		go task.Start(&wg, cache)
	}

	// Accept SIGINT or SIGTERM to gracefully shutdown the above periodic job
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Stop tasks
	for _, task := range *tasks {
		close(task.stop)
	}

	// Wait for all tasks to finish
	wg.Wait()
}
