/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"
)

// ParsedDownloaderConfig holds the parsed and validated configuration for a single downloader instance, used internally.
type ParsedDownloaderConfig struct {
	RpcType     string // "aria2c" or "transmission"
	RpcUrl      string // The fully constructed RPC URL (e.g., "http://localhost:6800/jsonrpc")
	Token       string // For aria2c authentication
	Username    string // For transmission authentication
	Password    string // For transmission authentication
	AutoCleanUp bool   // Whether to automatically clean up completed tasks
}

type Task struct {
	Name          string
	Downloaders   []ParsedDownloaderConfig
	FetchInterval time.Duration
	FeedUrls      []string
	parserConfig  *ParserConfig
	ctx           context.Context
}

// DownloadStatus represents the status of a download item
type DownloadStatus struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"` // "downloading", "seeding", "stopped", "error"
	IsFinished  bool    `json:"isFinished"`
	PercentDone float64 `json:"percentDone"`
	Downloader  string  `json:"downloader"` // "aria2c" or "transmission"
}

// RpcClient is the interface for both aria2c and transmission rpc clients.
type RpcClient interface {
	AddTorrent(uri string) error
	CleanUp()
	CloseRpc()
	GetActiveDownloads() ([]DownloadStatus, error) // New method to get download status
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

	t.fetchTorrents(cache, false)
	for {
		select {
		case <-ticker.C:
			t.fetchTorrents(cache, true)
			t.cleanUpTorrents()
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

	infoHashSet := t.getAllInfoHashes(cache)
	for _, feedUrl := range t.FeedUrls {
		parser := NewFeedParser(t.ctx, feedUrl, t.parserConfig)
		if parser == nil {
			continue
		}
		var processedItems map[string][]string
		if ignoreProcessed {
			processedItems = cache.Get(feedUrl)
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
				client.CloseRpc() // Close connection regardless of cleanup

				if err == nil {
					slog.Info("Successfully added torrent", "URL", torrent.URL, "downloader_type", dlConfig.RpcType)
					added = true
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

func (t *Task) cleanUpTorrents() {
	for _, dlConfig := range t.Downloaders {
		client, err := createRpcClientForConfig(t.ctx, dlConfig)
		if err != nil {
			slog.Warn("Failed to create RPC client for config, skipping", "type", dlConfig.RpcType, "error", err)
			continue
		}

		if dlConfig.AutoCleanUp { // Check the flag before cleaning up
			client.CleanUp()
		}
		client.CloseRpc() // Close connection regardless of cleanup
	}
}

func createRpcClientForConfig(ctx context.Context, cfg ParsedDownloaderConfig) (RpcClient, error) {
	var client RpcClient
	var err error

	switch cfg.RpcType {
	case "aria2c":
		// NewAria2c takes RpcUrl and Token
		client, err = NewAria2c(ctx, cfg.RpcUrl, cfg.Token)
		if err != nil && strings.Contains(cfg.RpcUrl, "ws://") || strings.Contains(cfg.RpcUrl, "wss://") {
			// Provide a more specific error if it's a WebSocket URL, as we explicitly disallow it in config parsing
			err = fmt.Errorf("aria2c WebSocket protocol is not supported: %w", err)
		}
	case "transmission":
		// NewTransmission takes RpcUrl, Username, and Password
		client, err = NewTransmission(ctx, cfg.RpcUrl, cfg.Username, cfg.Password)
	default:
		// This case should ideally not be reached due to validation in parseDownloaderConfig
		err = errors.New("unknown RpcType encountered in createRpcClientForConfig: " + cfg.RpcType)
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

func (s infoHashSet) add(hashes []string) {
	for _, h := range hashes {
		s[h] = struct{}{}
	}
}

func (s infoHashSet) contains(hash string) bool {
	_, ok := s[hash]
	return ok
}
