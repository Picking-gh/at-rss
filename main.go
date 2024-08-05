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

	config := NewConfig(opt.Config)

	client := NewAria2c(config.Server.Url, config.Server.Token)

	cache := NewCache()

	update := func() {
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
					log.Printf("Adding [%s] failed, %s", url, err)
				}
				time.Sleep(time.Second)
			}

			client.CleanUp()
		}
	}

	// Run once
	update()

	// Create cron job for periodic updating and fire
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

	// Accept SIGINT or SIGTERM for gracefully shutdown the above cron job
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	s.Shutdown()
}
