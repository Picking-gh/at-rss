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

type ServerConfig struct {
	RpcType  string // "aria2c" or "transmission"
	Url      string // for aria2c rpc
	Token    string // for aria2c rpc
	Host     string // for transmission rpc
	Port     uint16 // for transmission rpc
	Username string // for transmission rpc
	Password string // for transmission rpc
}

type Task struct {
	ServerConfig  ServerConfig
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

// doFetchTorrents contains the actual fetch logic
func (t *Task) doFetchTorrents(cache *Cache, ignoreProcessed bool) error {
	client, err := t.createRpcClient()
	if err != nil {
		slog.Warn("Failed to create RPC client", "type", t.ServerConfig.RpcType, "error", err)
		return err
	}
	defer func() {
		client.CleanUp()
		client.CloseRpc()
	}()

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
			if err := client.AddTorrent(torrent.URL); err != nil {
				// Mark item as unprocessed if it fails to add, so it's retried in the next fetchTorrents call
				slog.Error("Failed to add torrent",
					"URL", torrent.URL,
					"error", err)
				delete(newItems, guid)
			} else {
				// Avoid adding magnet links with duplicate infoHashes when processing multiple feeds.
				infoHashSet.add(torrent.InfoHashes)
				newItems[guid] = torrent.InfoHashes
			}
		}
		parser.RemoveExpiredItems(cache)
		cache.Set(feedUrl, newItems, false)
	}
	cache.Flush()
	return nil
}

// createRpcClient initializes the appropriate RPC client based on RpcType.
func (t *Task) createRpcClient() (RpcClient, error) {
	var client RpcClient
	var err error

	switch t.ServerConfig.RpcType {
	case "aria2c":
		client, err = NewAria2c(t.ctx, t.ServerConfig.Url, t.ServerConfig.Token)
	case "transmission":
		client, err = NewTransmission(t.ctx, t.ServerConfig.Host, t.ServerConfig.Port, t.ServerConfig.Username, t.ServerConfig.Password)
	default:
		err = errors.New("unknown RpcType: " + t.ServerConfig.RpcType)
	}

	return client, err
}

// infoHashSet is a memory-efficient set implementation for info hashes
type infoHashSet map[string]struct{}

func (t *Task) getAllInfoHashes(cache *Cache) infoHashSet {
	infoHashSet := make(infoHashSet)
	for _, items := range cache.data {
		for _, infoHashes := range items.Items { // Access the Items field of FeedCache
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
