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

// Feed keeps a RSS feed parsing configurations, parsed content and head item cache.
type Feed struct {
	*ParserConfig
	contents *gofeed.Feed
	cache    *Cache
}

// ParserConfig saves the parameters read from the configuration file.
type ParserConfig struct {
	FeedUrl string
	Include []string
	Exclude []string
	Trick   bool // Whether to apply the extractor to reconstruct the magnet link
	Pattern string
	Tag     string
	r       *regexp.Regexp
}

// getTagValue returns tag value in *gofeed.Item. For enclosure tag may apear multiple times, return []string for all kinds of tags.
// tagName is validated before that ensures no errors here.
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
		for i, item := range item.Enclosures {
			result[i] = item.URL
		}
		return result
	case "guid":
		return []string{item.GUID}
	}
	return []string{}
}

// NewFeedParser create a new Feed object
func NewFeedParser(ctx context.Context, pc *ParserConfig, cache *Cache) *Feed {
	ctx_, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	fp := gofeed.NewParser()
	contents, err := fp.ParseURLWithContext(pc.FeedUrl, ctx_)
	if err != nil {
		slog.Warn("Failed to fetch ["+pc.FeedUrl+"].", "err", err)
		return nil
	}
	return &Feed{pc, contents, cache}
}

// GetNewItems return all the new items in the RSS feed
func (f *Feed) GetNewItems() []*gofeed.Item {
	guid, err := f.cache.Get(f.FeedUrl)
	if err != nil {
		return f.contents.Items[:]
	}
	for i, item := range f.contents.Items {
		if item.GUID == guid {
			return f.contents.Items[:i]
		}
	}
	return f.contents.Items[:]
}

// GetNewTorrentURL return the url of all the new items in the RSS feed
func (f *Feed) GetNewTorrentURL() []string {
	urls := make([]string, 0)

	items := f.GetNewItems()
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

		if f.shouldSkipItem(strings.ToLower(title)) {
			continue
		}
		// Only print after finding the first item that meets the criteria to reduce unnecessary logs.
		if !hasExpectedItem {
			hasExpectedItem = true
			slog.Info("Fetching torrents from [" + f.FeedUrl + "]...")
		}

		slog.Info("Got " + item.Title)

		if f.Trick {
			// construct magnetic links
			for _, url := range getTagValue(item, f.Tag) {
				matchStrings := f.r.FindStringSubmatch(url)
				if len(matchStrings) < 2 {
					slog.Warn(f.Pattern + " matched no hash. Skipped.")
					continue
				}
				urls = append(urls, "magnet:?xt=urn:btih:"+matchStrings[1])
			}
		} else {
			// directly download torrent
			for _, enclosure := range item.Enclosures {
				if enclosure.Type == "application/x-bittorrent" {
					urls = append(urls, enclosure.URL)
				}
			}
		}
	}
	f.cache.Set(f.FeedUrl, items[0].GUID)
	return urls
}

// shouldSkipItem checks if an item should be skipped based on include and exclude filters
func (f *Feed) shouldSkipItem(title string) bool {
	// apply exclude filters
	// f.Exclude contain multiple strings, representing an AND relationship.
	// Each string is treated as a whole.
	for _, excludeKeyword := range f.Exclude {
		if strings.Contains(title, excludeKeyword) {
			return true
		}
	}

	// apply include filters
	// Empty f.Include means no filter.
	if len(f.Include) == 0 {
		return false
	}
	// f.Include contain multiple strings, representing an OR relationship.
	// Each string may contain comma separator, the separated parts are in an AND relationship.
	for _, includeKeywords := range f.Include {
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
