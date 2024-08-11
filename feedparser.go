/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/liuzl/gocc"
	"github.com/mmcdole/gofeed"
)

// Feed manages RSS feed parsing configurations, parsed content.
type Feed struct {
	*ParserConfig
	contents *gofeed.Feed
}

// ParserConfig holds the parameters read from the configuration file.
type ParserConfig struct {
	FeedUrl string
	Include []string
	Exclude []string
	Trick   bool // Whether to apply the extractor to reconstruct the magnet link
	Pattern string
	Tag     string
	r       *regexp.Regexp
}

// getTagValue returns tag value in *gofeed.Item. For enclosure tag, may appear multiple times; returns []string for all tags.
// tagName is validated before, ensuring no errors here.
func getTagValue(item *gofeed.Item, tagName string) []string {
	switch tagName {
	case "title":
		return []string{item.Title}
	case "link":
		return []string{item.Link}
	case "description":
		return []string{item.Description}
	case "enclosure":
		result := make([]string, len(item.Enclosures))
		for i, enclosure := range item.Enclosures {
			result[i] = enclosure.URL
		}
		return result
	case "guid":
		return []string{item.GUID}
	default:
		return nil
	}
}

// NewFeedParser creates a new Feed object.
func NewFeedParser(ctx context.Context, pc *ParserConfig) *Feed {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fp := gofeed.NewParser()
	contents, err := fp.ParseURLWithContext(pc.FeedUrl, ctx)
	if err != nil {
		slog.Warn("Failed to fetch feed URL", "url", pc.FeedUrl, "error", err)
		return nil
	}
	return &Feed{pc, contents}
}

// GetNewItems returns all new items in the RSS feed.
func (f *Feed) GetNewItems(cache *Cache) []*gofeed.Item {
	// Attempt to get GUIDs from cache
	guid, err := cache.Get(f.FeedUrl)
	if err != nil {
		return f.contents.Items
	}

	// Preallocate slice based on the number of items
	newItems := make([]*gofeed.Item, 0, len(f.contents.Items))

	// Find new items
	for _, item := range f.contents.Items {
		if _, found := guid[item.GUID]; !found {
			newItems = append(newItems, item)
		}
	}

	f.RemoveExpiredItems(cache)

	return newItems
}

// Pair stores torrent url and its index in []*gofeed.Item
type Pair struct {
	url   string
	index int
}

// GetNewTorrentURL returns the URLs of all new items in the RSS feed.
func (f *Feed) GetNewTorrentURL(items []*gofeed.Item) []Pair {
	urls := make([]Pair, 0, len(items))
	if len(items) == 0 {
		return urls
	}

	cc, _ := gocc.New("t2s") // Convert Traditional Chinese to Simplified Chinese
	hasExpectedItem := false
	for i, item := range items {
		var title string
		var err error
		if cc != nil {
			title, err = cc.Convert(item.Title)
			if err != nil {
				slog.Warn("Failed to convert title to simplified Chinese", "title", item.Title, "error", err)
				title = item.Title
			}
		} else {
			title = item.Title
		}

		if f.shouldSkipItem(strings.ToLower(title)) {
			continue
		}
		if !hasExpectedItem {
			hasExpectedItem = true
			slog.Info("Fetching torrents from", "url", f.FeedUrl)
		}

		slog.Info("Got item", "title", item.Title)

		if f.Trick {
			for _, url := range getTagValue(item, f.Tag) {
				matchStrings := f.r.FindStringSubmatch(url)
				if len(matchStrings) < 2 {
					slog.Warn("Pattern did not match any hash", "pattern", f.Pattern)
					continue
				}
				urls = append(urls, Pair{url: "magnet:?xt=urn:btih:" + matchStrings[1], index: i})
			}
		} else {
			for _, enclosure := range item.Enclosures {
				if enclosure.Type == "application/x-bittorrent" {
					urls = append(urls, Pair{url: enclosure.URL, index: i})
				}
			}
		}
	}
	return urls
}

// shouldSkipItem checks if an item should be skipped based on include and exclude filters.
func (f *Feed) shouldSkipItem(title string) bool {
	for _, excludeKeyword := range f.Exclude {
		if strings.Contains(title, excludeKeyword) {
			return true
		}
	}

	if len(f.Include) == 0 {
		return false
	}

	for _, includeKeywords := range f.Include {
		keywords := strings.Split(includeKeywords, ",")
		allMatch := true
		for _, keyword := range keywords {
			if !strings.Contains(title, strings.TrimSpace(keyword)) {
				allMatch = false
				break
			}
		}
		if allMatch {
			return false
		}
	}
	return true
}

// RemoveExpiredItems removes items from the cache that are not present in the feed
func (f *Feed) RemoveExpiredItems(cache *Cache) {
	// Create a set of feed GUIDs
	feedGuids := make(map[string]struct{}, len(f.contents.Items))
	for _, item := range f.contents.Items {
		feedGuids[item.GUID] = struct{}{}
	}

	// Access the cache for the specific feed URL
	cacheItems := cache.data[f.FeedUrl]

	// Remove cache items that are not in feedGuids
	for guid := range cacheItems {
		if _, found := feedGuids[guid]; !found {
			delete(cacheItems, guid)
		}
	}
}
