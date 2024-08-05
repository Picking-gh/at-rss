/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log"
	"strings"

	"github.com/liuzl/gocc"
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

	hasExpectedItem := false
	cc, _ := gocc.New("t2s") // "t2s" tradisional chinese -> simplified chinese
	for _, item := range items {
		// The filtering criteria ignore the distinction between traditional and simplified Chinese,
		// so here the item.Title is converted to simplified Chinese and compared with the keywords that have already been converted to simplified Chinese.
		var title string
		var err error
		if cc != nil {
			title, err = cc.Convert(item.Title)
		}
		if cc == nil || err != nil {
			title = item.Title
		}

		if a.shouldSkipItem(strings.ToLower(title)) {
			continue
		}
		// Only print after finding the first item that meets the criteria to reduce unnecessary logs.
		if !hasExpectedItem {
			hasExpectedItem = true
			log.Printf("Fetching torrents from [%s]...", a.Url)
		}

		log.Printf("Got %s", item.Title)

		if a.Trick {
			// construct magnetic link
			matchStrings := a.r.FindStringSubmatch(item.Link)
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
		if strings.Contains(title, excludeKeyword) {
			return true
		}
	}

	// apply include filters
	// Empty a.Include means no filter.
	if len(a.Include) == 0 {
		return false
	}
	// a.Include contain multiple strings, representing an OR relationship.
	// Each string may contain comma separator, the separated parts are in an AND relationship.
	for _, includeKeywords := range a.Include {
		hasMismatched := false
		for _, keyword := range strings.Split(includeKeywords, ",") {
			if !strings.Contains(title, strings.TrimSpace(keyword)) {
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
