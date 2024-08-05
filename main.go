/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-co-op/gocron/v2"
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

	var config *Config
	var configLock sync.Mutex
	var client *Aria2c

	// Watch config file for changes and reload.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()
	err = watcher.Add(opt.Config)
	if err != nil {
		log.Fatalf("Failed to watch file: %v", err)
	}
	loadConfig := func() {
		configLock.Lock()
		defer configLock.Unlock()
		config = NewConfig(opt.Config)
		client = NewAria2c(config.Server.Url, config.Server.Token)
	}
	go func() {
		for event := range watcher.Events {
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("Config file changed. Reloading...")
				loadConfig()
			}
		}
	}()

	loadConfig()
	cache := NewCache()

	// Parse feeds and fire downloading on current config.
	// In update progress config may change.
	update := func() {
		configLock.Lock()
		configNow := config
		clientNow := client
		configLock.Unlock()

		for i := range configNow.Feeds {
			feed := &configNow.Feeds[i]
			aggregator := NewAggregator(feed, cache)
			if aggregator == nil {
				continue
			}

			urls := aggregator.GetNewTorrentURL()
			for _, url := range urls {
				err := clientNow.Add(url)
				if err != nil {
					log.Printf("Adding [%s] failed, %v", url, err)
				}
				time.Sleep(time.Second)
			}

			clientNow.CleanUp()
		}
	}

	// Run once
	update()

	// Create periodic job to update
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal("Unable to create new scheduler")
	}

	_, err = s.NewJob(
		gocron.DurationJob(
			time.Duration(config.UpdateInterval)*time.Minute,
		),
		gocron.NewTask(update),
	)
	if err != nil {
		log.Fatal("Unable to create periodic tasks")
	}
	s.Start()

	// Accept SIGINT or SIGTERM to gracefully shutdown the above periodic job
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	s.Shutdown()
}
