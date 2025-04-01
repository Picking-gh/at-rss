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
	cc      *gocc.OpenCC // Cached Chinese converter
}

// ParserConfig holds the parameters read from the configuration file.
type ParserConfig struct {
	Include []string
	Exclude []string
	Trick   bool // Whether to apply the extractor to reconstruct the magnet link
	Pattern string
	Tag     string
	r       *regexp.Regexp // Pre-compiled regex
}

// NewParserConfig creates a new ParserConfig with pre-compiled regex
func NewParserConfig(include, exclude []string, trick bool, pattern, tag string) (*ParserConfig, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &ParserConfig{
		Include: include,
		Exclude: exclude,
		Trick:   trick,
		Pattern: pattern,
		Tag:     tag,
		r:       r,
	}, nil
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

	cc, _ := gocc.New("t2s") // Initialize converter once
	return &Feed{pc, contents, url, ctx, cc}
}

// ProcessFeedItem processes a single feed item to extract relevant torrent URLs.
// It returns a TorrentInfo object containing the URL and related info hashes.
func (f *Feed) ProcessFeedItem(item *gofeed.Item, ignoredInfoHashSet map[string]struct{}) *TorrentInfo {
	// Apply include and exclude filters on the title
	var title string
	rawTitle := html.UnescapeString(item.Title)
	if f.cc != nil {
		var err error
		title, err = f.cc.Convert(rawTitle)
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
			infoHashes, _ := parseURI(f.ctx, enclosureURL)
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
	if f.matchesExcludeFilters(title) {
		return true
	}
	return !f.matchesIncludeFilters(title)
}

// matchesExcludeFilters checks if title matches any exclude filter
func (f *Feed) matchesExcludeFilters(title string) bool {
	for _, keywords := range f.Exclude {
		if allKeywordsMatch(title, keywords) {
			return true
		}
	}
	return false
}

// matchesIncludeFilters checks if title matches include filters
func (f *Feed) matchesIncludeFilters(title string) bool {
	if len(f.Include) == 0 {
		return true // No include filters means include all
	}
	for _, keywords := range f.Include {
		if allKeywordsMatch(title, keywords) {
			return true
		}
	}
	return false
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

// parseURI parses a URI and returns all infohashes, handling both magnet and torrent URIs
func parseURI(ctx context.Context, uri string) ([]string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "magnet":
		return parseMagnetURI(uri)
	case "http", "https":
		return parseTorrentURI(ctx, uri)
	default:
		return nil, errors.New("unsupported URI scheme")
	}
}

// parseMagnetURI extracts infohashes from magnet URI
func parseMagnetURI(uri string) ([]string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
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

// parseTorrentURI downloads and parses torrent file to get infohash
func parseTorrentURI(ctx context.Context, uri string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
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
