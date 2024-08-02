/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log"
	"os"

	"github.com/jasonlvhit/gocron"
	"github.com/jessevdk/go-flags"
)

type options struct {
	Config string `short:"c" long:"conf" description:"Config file" default:"/etc/transmission-rss.conf"`
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
		for _, feed := range config.Feeds {
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
			}
		}
	}

	// Run now
	update()

	// Schedule
	gocron.Every(config.UpdateInterval).Minutes().Do(update)

	<-gocron.Start()
}
