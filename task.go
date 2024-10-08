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

// Start begins executing the task at regular intervals.
func (t *Task) Start(ctx context.Context, cache *Cache) {
	ticker := time.NewTicker(t.FetchInterval)
	defer ticker.Stop()
	t.ctx = ctx

	// Fetch torrents initially and then repeatedly at intervals
	t.fetchTorrents(cache)
	for {
		select {
		case <-ticker.C:
			t.fetchTorrents(cache)
		case <-t.ctx.Done():
			return
		}
	}
}

// fetchTorrents retrieves torrents via the appropriate RPC client.
func (t *Task) fetchTorrents(cache *Cache) {
	client, err := t.createRpcClient()
	if err != nil {
		slog.Warn("Failed to create RPC client", "rpcType", t.ServerConfig.RpcType, "err", err)
		return
	}
	defer func() {
		client.CleanUp()
		client.CloseRpc()
	}()

	// infoHashMap keeps track of the hashes of magnet links added
	infoHashMap := cache.Get("infoHash")
	for _, feedUrl := range t.FeedUrls {
		parser := NewFeedParser(t.ctx, feedUrl, t.parserConfig)
		if parser == nil {
			continue
		}
		processedItems := cache.Get(feedUrl) // Items processed before
		newItems := parser.GetGUIDSet()

		for _, item := range parser.Content.Items {
			guid := html.UnescapeString(item.GUID)
			if _, alreadyProcessed := processedItems[guid]; alreadyProcessed {
				continue
			}
			torrent := parser.ProcessFeedItem(item, infoHashMap)
			if torrent == nil {
				continue
			}
			if err := client.AddTorrent(torrent.URL); err != nil {
				// Mark item as unprocessed if it fails to add, so it's retried in the next fetchTorrents call
				slog.Warn("Failed to add torrent", "URL", torrent.URL, "err", err)
				delete(newItems, guid)
			} else {
				// Avoid adding magnet links with duplicate infoHashes when processing multiple feeds.
				// Store added magnet links' infoHashes
				for _, infoHash := range torrent.InfoHashes {
					infoHashMap[infoHash] = time.Now().Unix()
				}
			}
		}
		parser.RemoveExpiredItems(cache)
		cache.Set(feedUrl, newItems)
	}
	cache.Set("infoHash", infoHashMap)
	cache.Flush()
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
