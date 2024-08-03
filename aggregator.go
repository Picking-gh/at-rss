/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log"
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"
)

// Aggregator is a RSS aggregator object
type Aggregator struct {
	*Feed
	contents *gofeed.Feed
	cache    *Cache
}

// NewAggregator create a new Aggregator object
func NewAggregator(feed *Feed, cache *Cache) *Aggregator {
	fp := gofeed.NewParser()
	contents, err := fp.ParseURL(feed.Url)
	if err != nil {
		log.Printf("Fetching [%s] failed, %s", feed.Url, err)
		return nil
	}
	return &Aggregator{feed, contents, cache}
}

// GetNewItems return all the new items in the RSS feed
func (a *Aggregator) GetNewItems() []*gofeed.Item {
	guid, err := a.cache.Get(a.Url)
	if err != nil {
		return a.contents.Items[:]
	}
	for i, item := range a.contents.Items {
		if item.GUID == guid {
			return a.contents.Items[:i]
		}
	}
	return a.contents.Items[:]
}

// GetNewTorrentURL return the url of all the new items in the RSS feed
func (a *Aggregator) GetNewTorrentURL() []string {
	urls := make([]string, 0)

	items := a.GetNewItems()
	if len(items) == 0 {
		return urls
	}

	// compile trick pattern
	var r *regexp.Regexp = nil
	var err error = nil
	if a.Trick {
		r, err = regexp.Compile(a.Pattern)
		if err != nil {
			log.Println("Pattern invalid. Skip the feed.")
			return urls
		}
	}

	log.Printf("Fetching torrents from [%s]...", a.Url)

	for _, item := range items {
		if a.shouldSkipItem(strings.ToLower(item.Title)) {
			continue
		}

		log.Printf("Got %s", item.Title)

		if a.Trick {
			// construct magnetic link
			matchStrings := r.FindStringSubmatch(item.Link)
			if len(matchStrings) < 2 {
				continue
			}
			url := "magnet:?xt=urn:btih:" + matchStrings[1]
			urls = append(urls, url)
		} else {
			// directly download torrent
			for _, enclosure := range item.Enclosures {
				if enclosure.Type == "application/x-bittorrent" {
					urls = append(urls, enclosure.URL)
				}
			}
		}
	}
	a.cache.Set(a.Url, items[0].GUID)
	return urls
}

// shouldSkipItem checks if an item should be skipped based on include and exclude filters
func (a *Aggregator) shouldSkipItem(title string) bool {
	// apply exclude filters
	// a.Exclude contain multiple strings, representing an AND relationship.
	// Each string is treated as a whole.
	for _, excludeKeyword := range a.Exclude {
		if strings.Contains(title, strings.ToLower(strings.TrimSpace(excludeKeyword))) {
			return true
		}
	}

	// apply include filters
	// a.Include contain multiple strings, representing an OR relationship.
	// Each string may contain comma separator, the separated parts are in an AND relationship.
	// Empty a.Include means no filter.
	if len(a.Include) == 0 {
		return false
	}
	for _, includeKeywords := range a.Include {
		hasMismatched := false
		for _, keyword := range strings.Split(includeKeywords, ",") {
			if !strings.Contains(title, strings.ToLower(strings.TrimSpace(keyword))) {
				hasMismatched = true
				break
			}
		}
		if !hasMismatched {
			return false
		}
	}
	return true
}
