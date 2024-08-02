/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log"

	"github.com/mmcdole/gofeed"
)

// Aggregator is a RSS aggregator object
type Aggregator struct {
	url   string
	feed  *gofeed.Feed
	cache *Cache
}

// NewAggregator create a new Aggregator object
func NewAggregator(url string, cache *Cache) *Aggregator {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		log.Printf("Fetching [%s] failed, %s", url, err)
		return nil
	}
	return &Aggregator{url, feed, cache}
}

// GetNewItems return all the new items in the RSS feed
func (a *Aggregator) GetNewItems() []*gofeed.Item {
	guid, err := a.cache.Get(a.url)
	if err != nil {
		return a.feed.Items[:]
	}
	for i, item := range a.feed.Items {
		if item.GUID == guid {
			return a.feed.Items[:i]
		}
	}
	return a.feed.Items[:]
}

// GetNewTorrentURL return the url of all the new items in the RSS feed
func (a *Aggregator) GetNewTorrentURL() []string {
	urls := make([]string, 0)

	items := a.GetNewItems()
	if len(items) == 0 {
		return urls
	}

	log.Printf("Fetching [%s] got %d new item(s)\n", a.url, len(items))

	for _, item := range items {
		log.Println(item.Title)
		for _, enclosure := range item.Enclosures {
			if enclosure.Type == "application/x-bittorrent" {
				urls = append(urls, enclosure.URL)
			}
		}
	}
	a.cache.Set(a.url, items[0].GUID)
	return urls
}
