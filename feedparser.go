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
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/liuzl/gocc"
	"github.com/mmcdole/gofeed"
)

const btihPrefix = "urn:btih:"

// Feed manages RSS feed parsing configurations and parsed content.
type Feed struct {
	*ParserConfig
	Content *gofeed.Feed
	URL     string // Feed URL
	ctx     context.Context
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
	InfoHashes []string // List of infohashes found in the item
}

// NewFeedParser creates a new Feed object for the specified URL.
func NewFeedParser(ctx context.Context, url string, pc *ParserConfig) *Feed {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fp := gofeed.NewParser()
	contents, err := fp.ParseURLWithContext(url, ctxWithTimeout)
	if err != nil {
		slog.Warn("Failed to fetch feed URL", "url", url, "error", err)
		return nil
	}
	return &Feed{pc, contents, url, ctx}
}

// ProcessFeedItem processes a single feed item to extract relevant torrent URLs.
// It returns a TorrentInfo object containing the URL and related info hashes.
func (f *Feed) ProcessFeedItem(item *gofeed.Item, ignoredInfoHashSet map[string]struct{}) *TorrentInfo {
	// Apply include and exclude filters on the title
	cc, _ := gocc.New("t2s") // Convert Traditional Chinese to Simplified Chinese
	var title string
	rawTitle := html.UnescapeString(item.Title)
	if cc != nil {
		var err error
		title, err = cc.Convert(rawTitle)
		if err != nil {
			slog.Warn("Failed to convert title to simplified Chinese", "title", rawTitle, "error", err)
			title = rawTitle
		}
	} else {
		title = rawTitle
	}
	if f.shouldSkipItem(strings.ToLower(title)) {
		return nil
	}

	slog.Info("Processing item", "title", rawTitle, "url", f.URL)

	if f.Trick {
		for _, value := range getTagValue(item, f.Tag) {
			matchStrings := f.r.FindStringSubmatch(value)
			if len(matchStrings) < 2 {
				slog.Warn("Pattern did not match any hash", "pattern", f.Pattern)
				continue
			}
			// Avoid adding magnet links with duplicate infoHashes when processing multiple feeds.
			infoHash, err := regulateInfoHash(matchStrings[1])
			if err != nil {
				slog.Warn("Matched infoHash not valid", "error", err)
				continue
			}
			if _, exists := ignoredInfoHashSet[infoHash]; exists {
				continue
			}
			url := "magnet:?xt=" + btihPrefix + infoHash
			slog.Info("Added URL", "url", url)
			return &TorrentInfo{URL: url, InfoHashes: []string{infoHash}}
		}
	} else {
		for _, enclosure := range item.Enclosures {
			if enclosure.Type != "application/x-bittorrent" {
				continue
			}
			// Prevent adding magnet links with duplicate infoHashes when processing multiple feeds.
			// For non-magnet links, attempt to obtain the infoHash from the downloaded torrent file (supports HTTP only).
			enclosureURL := html.UnescapeString(enclosure.URL)
			infoHashes, err := parseMagnetURI(enclosureURL)
			if err != nil {
				infoHashes, _ = parseTorrentURIWithTimeout(f.ctx, enclosureURL)
			}
			// If any error occurs, infoHashes slice is empty. In this case, do not apply infoHash filter.
			if len(infoHashes) == 0 {
				slog.Info("Added URL", "url", enclosureURL)
				return &TorrentInfo{URL: enclosureURL, InfoHashes: nil}
			}
			for _, infoHash := range infoHashes {
				// Add to download link list if at least one infoHash hasn't been downloaded.
				if _, exists := ignoredInfoHashSet[infoHash]; !exists {
					slog.Info("Added URL", "url", enclosureURL)
					return &TorrentInfo{URL: enclosureURL, InfoHashes: infoHashes}
				}
			}
		}
	}
	return nil
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

// RemoveExpiredItems removes items from the cache that are not present in the feed.
func (f *Feed) RemoveExpiredItems(cache *Cache) {
	cache.RemoveNotIn(f.URL, f.GetGUIDSet())
}

// GetGUIDSet creates a set of feed GUIDs.
func (f *Feed) GetGUIDSet() map[string][]string {
	feedGUIDs := make(map[string][]string, len(f.Content.Items))
	for _, item := range f.Content.Items {
		feedGUIDs[html.UnescapeString(item.GUID)] = nil
	}
	return feedGUIDs
}

// getTagValue returns tag values in *gofeed.Item. For enclosure tags, it may appear multiple times; returns []string for all tags.
func getTagValue(item *gofeed.Item, tagName string) []string {
	switch tagName {
	case "title":
		return []string{html.UnescapeString(item.Title)}
	case "link":
		return []string{html.UnescapeString(item.Link)}
	case "description":
		return []string{html.UnescapeString(item.Description)}
	case "enclosure":
		result := make([]string, len(item.Enclosures))
		for i, enclosure := range item.Enclosures {
			result[i] = html.UnescapeString(enclosure.URL)
		}
		return result
	case "guid":
		return []string{html.UnescapeString(item.GUID)}
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

// parseMagnetURI parses a URI and returns all infohashes as hex strings if the URI is magnet-formatted.
// If URI is not a magnet link or is not valid, returns an error.
func parseMagnetURI(uri string) ([]string, error) {
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

	return hashes, nil
}

// regulateInfoHash decodes the infoHash from the string and returns its hex representation.
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

// parseTorrentURIWithTimeout downloads a torrent file from the specified URI using an HTTP GET request
// with a context-based timeout. It parses the torrent file's metadata and returns the info hash as a hex string.
// If the request fails or the torrent file cannot be parsed, it returns an error.
func parseTorrentURIWithTimeout(ctx context.Context, uri string) ([]string, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxWithTimeout, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	metaInfo, err := metainfo.Load(resp.Body)
	if err != nil {
		return nil, err
	}

	return []string{metaInfo.HashInfoBytes().HexString()}, nil
}
