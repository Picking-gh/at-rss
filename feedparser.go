/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/liuzl/gocc"
	"github.com/mmcdole/gofeed"
)

const btihPrefix = "urn:btih:"

// Feed manages RSS feed parsing configurations, parsed content.
type Feed struct {
	*ParserConfig
	Contents *gofeed.Feed
	URL      string // feed URL
}

// ParserConfig holds the parameters read from the configuration file.
type ParserConfig struct {
	Include []string
	Exclude []string
	Trick   bool // Whether to apply the extractor to reconstruct the magnet link
	Pattern string
	Tag     string
	r       *regexp.Regexp
}

// TorrentInfo represents a single torrent or magnet link found in a feed item.
type TorrentInfo struct {
	URL        string   // URL of the .torrent file or magnet link
	Index      int      // Index of the item in the feed
	InfoHashes []string // List of infohashes found in the item
}

// NewFeedParser creates a new Feed object of url.
func NewFeedParser(ctx context.Context, url string, pc *ParserConfig) *Feed {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fp := gofeed.NewParser()
	contents, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		slog.Warn("Failed to fetch feed URL", "url", url, "error", err)
		return nil
	}
	return &Feed{pc, contents, url}
}

// GetNewItems returns all new items in the RSS feed.
// The cache logs all added items to avoid parsing the same item multiple times.
func (f *Feed) GetNewItems(cache *Cache) []*gofeed.Item {
	// Attempt to get GUIDs of added items from cache
	guid, err := cache.Get(f.URL)
	if err != nil {
		return f.Contents.Items
	}

	// Preallocate slice based on the number of items
	newItems := make([]*gofeed.Item, 0, len(f.Contents.Items))

	// Find new items
	for _, item := range f.Contents.Items {
		if _, found := guid[item.GUID]; !found {
			newItems = append(newItems, item)
		}
	}

	f.RemoveExpiredItems(cache)

	return newItems
}

// GetNewTorrents returns the URLs of all new items in the RSS feed.
// The infoHashSet logs infoHash of all added magnet links to avoid adding the same link multiple times.
func (f *Feed) GetNewTorrents(items []*gofeed.Item, infoHashSet map[string]struct{}) []TorrentInfo {
	urls := make([]TorrentInfo, 0, len(items))
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
				slog.Warn("Failed to convert title to simplified Chinese.", "title", item.Title, "error", err)
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
			slog.Info("Fetching torrents from feed.", "url", f.URL)
		}

		slog.Info("Got item", "title", item.Title)

		if f.Trick {
			for _, url := range getTagValue(item, f.Tag) {
				matchStrings := f.r.FindStringSubmatch(url)
				if len(matchStrings) < 2 {
					slog.Warn("Pattern did not match any hash.", "pattern", f.Pattern)
					continue
				}
				// Avoid adding magnet links with duplicate infoHashes when processing multiple feeds.
				infoHash, err := regulateInfoHash(matchStrings[1])
				if err != nil {
					slog.Warn("Matched infoHash not valide", "err", err)
					continue
				}
				if _, exist := infoHashSet[infoHash]; exist {
					continue
				}
				urls = append(urls, TorrentInfo{URL: "magnet:?xt=" + btihPrefix + infoHash, Index: i, InfoHashes: []string{infoHash}})
			}
		} else {
			for _, enclosure := range item.Enclosures {
				if enclosure.Type != "application/x-bittorrent" {
					continue
				}
				// Avoid adding magnet links with duplicate infoHashes when processing multiple feeds.
				// If it's not a magnet link (including cases where it's a magnet link but malformed), let the BT client decide what to do.
				// Do not attempt to obtain the infoHash from the torrent at this time (todo).
				if infoHashes, err := parseMagnetUri(enclosure.URL); err != nil {
					urls = append(urls, TorrentInfo{URL: enclosure.URL, Index: i, InfoHashes: nil})
				} else {
					for _, infoHash := range infoHashes {
						// As long as there is at least one infoHash that hasn't been downloaded, add it to the download link list.
						if _, exist := infoHashSet[infoHash]; !exist {
							urls = append(urls, TorrentInfo{URL: enclosure.URL, Index: i, InfoHashes: infoHashes})
							break
						}
					}
				}
			}
		}
	}
	return urls
}

// shouldSkipItem checks if an item should be skipped based on include and exclude filters.
func (f *Feed) shouldSkipItem(title string) bool {
	// Check if all exclude keywords are present; if so, skip the item
	for _, excludeKeywords := range f.Exclude {
		if allKeywordsMatch(title, excludeKeywords) {
			return true
		}
	}

	// If there are no include keywords, do not skip the item
	if len(f.Include) == 0 {
		return false
	}

	// Check if all include keywords are present; if so, do not skip the item
	for _, includeKeywords := range f.Include {
		if allKeywordsMatch(title, includeKeywords) {
			return false
		}
	}

	// If none of the include keywords match, skip the item
	return true
}

// RemoveExpiredItems removes items from the cache that are not present in the feed
func (f *Feed) RemoveExpiredItems(cache *Cache) {
	feedGuids := f.GetGUIDSet()

	// Access the cache for the specific feed URL
	cacheItems := cache.data[f.URL]

	// Remove cache items that are not in feedGuids
	for guid := range cacheItems {
		if _, found := feedGuids[guid]; !found {
			delete(cacheItems, guid)
		}
	}
}

// getItemGuidSet creates a set of feed GUIDs
func (f *Feed) GetGUIDSet() map[string]struct{} {
	feedGuids := make(map[string]struct{}, len(f.Contents.Items))
	for _, item := range f.Contents.Items {
		feedGuids[item.GUID] = struct{}{}
	}
	return feedGuids
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

// allKeywordsMatch checks if all keywords in a comma-separated list are present in the title.
func allKeywordsMatch(title, keywords string) bool {
	keywordList := strings.Split(keywords, ",")
	for _, keyword := range keywordList {
		if !strings.Contains(title, strings.TrimSpace(keyword)) {
			return false
		}
	}
	return true
}

// parseMagnetUri parses a URI and returns all infohashes as hex strings if the URI is magnet-formatted.
func parseMagnetUri(uri string) ([]string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "magnet" {
		return nil, errors.New("not a magnet link")
	}

	q := u.Query()
	var hashes []string

	for _, xt := range q["xt"] {
		if !strings.HasPrefix(xt, btihPrefix) {
			continue
		}

		encoded := strings.TrimPrefix(xt, btihPrefix)
		hash, err := regulateInfoHash(encoded)
		if err != nil {
			continue
		}

		hashes = append(hashes, hash)
	}

	if len(hashes) == 0 {
		return nil, errors.New("no valid urn:btih found")
	}

	return hashes, nil
}

// regulateInfoHash decodes the infoHash from the s string and returns its hex representation.
func regulateInfoHash(s string) (string, error) {
	var decoded []byte
	var err error

	switch len(s) {
	case 40:
		decoded, err = hex.DecodeString(s)
	case 32:
		decoded, err = base32.StdEncoding.DecodeString(s)
	default:
		return "", errors.New("invalid urn:btih length")
	}

	if err != nil || len(decoded) != 20 {
		return "", errors.New("invalid urn:btih encoding")
	}

	return hex.EncodeToString(decoded), nil
}
