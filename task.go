/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"errors"
	"html"
	"log/slog"
	"time"
)

// ParsedDownloaderConfig holds the parsed and validated configuration for a single downloader instance, used internally.
type ParsedDownloaderConfig struct {
	RpcType  string // "aria2c" or "transmission"
	Url      string // for aria2c rpc
	Token    string // for aria2c rpc
	Host     string // for transmission rpc
	Port     uint16 // for transmission rpc
	Username string // for transmission rpc
	Password string // for transmission rpc
}

type Task struct {
	Downloaders   []ParsedDownloaderConfig
	FetchInterval time.Duration
	FeedUrls      []string
	parserConfig  *ParserConfig
	ctx           context.Context
}

// RpcClient is the interface for both aria2c and transmission rpc clients.
type RpcClient interface {
	AddTorrent(uri string) error
	CleanUp()
	CloseRpc()
}

// Start begins executing the task at regular intervals defined by FetchInterval.
// It runs an initial fetch immediately, then continues fetching at each interval.
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - cache: Cache instance for tracking processed items
//
// The function will exit when the context is cancelled.
func (t *Task) Start(ctx context.Context, cache *Cache) {
	ticker := time.NewTicker(t.FetchInterval)
	defer ticker.Stop()
	t.ctx = ctx

	// Fetch torrents initially and then repeatedly at intervals
	// The initial invoking does not ignore processed items. In this case, configure may have been changed, and shall check processed items to apply new filters
	// The repeated invokings ignore processed items. In this case, configure is kept unchanged.
	t.fetchTorrents(cache, false)
	for {
		select {
		case <-ticker.C:
			t.fetchTorrents(cache, true)
		case <-t.ctx.Done():
			return
		}
	}
}

// fetchTorrents retrieves torrents via the appropriate RPC client with retry mechanism.
func (t *Task) fetchTorrents(cache *Cache, ignoreProcessed bool) {
	const maxRetries = 3
	var lastErr error

	for i := range maxRetries {
		if err := t.doFetchTorrents(cache, ignoreProcessed); err != nil {
			lastErr = err
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		return
	}

	slog.Error("Failed to fetch torrents after retries",
		"retries", maxRetries,
		"error", lastErr)
}

// doFetchTorrents contains the actual fetch logic, attempting downloaders sequentially.
func (t *Task) doFetchTorrents(cache *Cache, ignoreProcessed bool) error {

	// infoHashSet keeps track of the hashes of magnet links added
	infoHashSet := t.getAllInfoHashes(cache)
	for _, feedUrl := range t.FeedUrls {
		parser := NewFeedParser(t.ctx, feedUrl, t.parserConfig)
		if parser == nil {
			continue
		}
		var processedItems map[string][]string
		if ignoreProcessed {
			processedItems = cache.Get(feedUrl) // Items processed before
		}
		newItems := parser.GetGUIDSet()

		for _, item := range parser.Content.Items {
			guid := html.UnescapeString(item.GUID)
			if ignoreProcessed {
				if _, alreadyProcessed := processedItems[guid]; alreadyProcessed {
					continue
				}
			}
			torrent := parser.ProcessFeedItem(item, infoHashSet.contains)
			if torrent == nil {
				continue
			}
			added := false
			var lastAddErr error
			for _, dlConfig := range t.Downloaders {
				client, err := createRpcClientForConfig(t.ctx, dlConfig)
				if err != nil {
					slog.Warn("Failed to create RPC client for config, skipping", "type", dlConfig.RpcType, "error", err)
					lastAddErr = err
					continue
				}

				err = client.AddTorrent(torrent.URL)
				client.CleanUp()  // Clean up immediately after use
				client.CloseRpc() // Close immediately after use

				if err == nil {
					slog.Info("Successfully added torrent", "URL", torrent.URL, "downloader_type", dlConfig.RpcType)
					added = true
					// Avoid adding magnet links with duplicate infoHashes when processing multiple feeds.
					infoHashSet.add(torrent.InfoHashes)
					newItems[guid] = torrent.InfoHashes
					break // Success, move to the next torrent item
				} else {
					slog.Warn("Failed to add torrent with downloader",
						"URL", torrent.URL,
						"downloader_type", dlConfig.RpcType,
						"error", err)
					lastAddErr = err // Keep track of the last error
				}
			}

			if !added {
				// Mark item as unprocessed if all downloaders failed
				slog.Error("Failed to add torrent with all downloaders",
					"URL", torrent.URL,
					"last_error", lastAddErr) // Log the last encountered error
				delete(newItems, guid)
			}
		}
		parser.RemoveExpiredItems(cache)
		cache.Set(feedUrl, newItems, false)
	}
	cache.Flush()
	return nil
}

// createRpcClientForConfig initializes the appropriate RPC client based on a specific ParsedDownloaderConfig.
// This is now a standalone function, not a method on Task.
func createRpcClientForConfig(ctx context.Context, cfg ParsedDownloaderConfig) (RpcClient, error) {
	var client RpcClient
	var err error

	switch cfg.RpcType {
	case "aria2c":
		client, err = NewAria2c(ctx, cfg.Url, cfg.Token)
	case "transmission":
		client, err = NewTransmission(ctx, cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	default:
		err = errors.New("unknown RpcType: " + cfg.RpcType)
	}

	return client, err
}

// infoHashSet is a memory-efficient set implementation for info hashes
type infoHashSet map[string]struct{}

func (t *Task) getAllInfoHashes(cache *Cache) infoHashSet {
	infoHashSet := make(infoHashSet)
	for _, feedCache := range cache.data {
		for _, infoHashes := range feedCache.Items { // Iterate through Items map in FeedCache
			for _, infoHash := range infoHashes {
				infoHashSet[infoHash] = struct{}{}
			}
		}
	}
	return infoHashSet
}

// add adds info hashes to the set
func (s infoHashSet) add(hashes []string) {
	for _, h := range hashes {
		s[h] = struct{}{}
	}
}

// contains checks if a hash exists in the set
func (s infoHashSet) contains(hash string) bool {
	_, ok := s[hash]
	return ok
}
