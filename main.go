/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	config, err := NewConfig(opt.Config)
	if err != nil {
		os.Exit(1)
	}
	cache, err := NewCache()
	if err != nil {
		os.Exit(1)
	}

	// Parse feeds and fire downloading on current config.
	// In update progress config may change.
	update := func() {
		client, err := NewAria2c(config.Server.Url, config.Server.Token)
		if err != nil {
			slog.Warn("Failed to connect to aria2c rpc.", "err", err)
			return
		}

		for i := range config.Feeds {
			feed := &config.Feeds[i]
			aggregator := NewAggregator(feed, cache)
			if aggregator == nil {
				continue
			}

			urls := aggregator.GetNewTorrentURL()
			for _, url := range urls {
				err := client.Add(url)
				if err != nil {
					slog.Warn("Failed to add ["+url+"].", "err", err)
				}
			}

		}

		client.CleanUp()
		client.Close()
	}

	// Run once
	update()

	// Create periodic job to update
	s, err := gocron.NewScheduler()
	if err != nil {
		slog.Error("Unable to create new scheduler.", "err", err)
		os.Exit(1)
	}

	_, err = s.NewJob(
		gocron.DurationJob(
			time.Duration(config.UpdateInterval)*time.Minute,
		),
		gocron.NewTask(update),
	)
	if err != nil {
		slog.Error("Unable to create periodic tasks.", "err", err)
		os.Exit(1)
	}
	s.Start()

	// Accept SIGINT or SIGTERM to gracefully shutdown the above periodic job
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	s.Shutdown()
}
