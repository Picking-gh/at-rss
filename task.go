/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type Task struct {
	Server struct {
		RpcType string // "aria2c" or "transmission"
		Url     string // for aria2c rpc
		Token   string // for aria2c rpc
		Host    string // for transmission rpc
		Port    uint16 // for transmission rpc
		User    string // for transmission rpc
		Pswd    string // for transmission rpc
	}
	FetchInterval time.Duration // Changed to time.Duration for better time handling
	FeedUrls      []string
	pc            *ParserConfig
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
	client, err := t.createClient()
	if err != nil {
		slog.Warn("Failed to create RPC client", "rpcType", t.Server.RpcType, "err", err)
		return
	}
	defer func() {
		client.CleanUp()
		client.CloseRpc()
	}()

	// hashSet keeps the hashes of a magnet link that is added
	infoHashSet := make(map[string]struct{})
	for _, url := range t.FeedUrls {
		parser := NewFeedParser(t.ctx, url, t.pc)
		if parser == nil {
			return
		}
		items := parser.GetNewItems(cache)
		torrents := parser.GetNewTorrents(items, infoHashSet)
		addedItems := parser.GetGUIDSet()
		for _, t := range torrents {
			if err := client.AddTorrent(t.URL); err != nil {
				slog.Warn("Failed to add torrent", "URL", t.URL, "err", err)
				delete(addedItems, items[t.Index].GUID)
			} else {
				// Avoid adding magnet links with duplicate infoHashes when processing multiple feeds.
				// Store added megnet links
				for _, infoHash := range t.InfoHashes {
					infoHashSet[infoHash] = struct{}{}
				}
			}
		}
		cache.Set(url, addedItems)
	}
}

// createClient initializes the appropriate RPC client based on RpcType.
func (t *Task) createClient() (RpcClient, error) {
	var client RpcClient
	var err error

	switch t.Server.RpcType {
	case "aria2c":
		client, err = NewAria2c(t.ctx, t.Server.Url, t.Server.Token)
	case "transmission":
		client, err = NewTransmission(t.ctx, t.Server.Host, t.Server.Port, t.Server.User, t.Server.Pswd)
	default:
		err = errors.New("unknown rpcType: " + t.Server.RpcType)
	}

	return client, err
}
