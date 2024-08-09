/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log/slog"
	"sync"
	"time"
)

type Task struct {
	Server struct {
		RpcType string //"aria2c" or "transmission"
		Url     string // for aria2c
		Token   string // for aria2c
		Host    string // for transmission rpc
		Port    uint16 // for transmission rpc
		User    string // for transmission rpc
		Pswd    string // for transmission rpc
	}
	FetchInterval uint64
	tp            *TorrentParser
	stop          chan bool
}

// RpcClient is the interface for both aria2c and transmission rpc client.
type RpcClient interface {
	AddTorrent(uri string) error
	CleanUp()
	Close()
}

// Start begins executing the task at regular intervals
func (t *Task) Start(wg *sync.WaitGroup, cache *Cache) {
	defer wg.Done()
	ticker := time.NewTicker(time.Duration(t.FetchInterval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.FetchTorrents(cache)
		case <-t.stop:
			return
		}
	}
}

// FetchTorrents gets torrents via rpc client of proper RpcType
func (t *Task) FetchTorrents(cache *Cache) {
	var client RpcClient
	var err error

	switch t.Server.RpcType {
	case "aria2c":
		client, err = NewAria2c(t.Server.Url, t.Server.Token)
		if err != nil {
			slog.Warn("Failed to connect to aria2c rpc.", "err", err)
			return
		}
	case "transmission":
		client, err = NewTransmission(t.Server.Host, t.Server.Port, t.Server.User, t.Server.Pswd)
		if err != nil {
			slog.Warn("Failed to connect to transmission rpc.", "err", err)
			return
		}
	}
	defer func() {
		client.CleanUp()
		client.Close()
	}()

	parser := NewFeedParser(t.tp, cache)
	if parser == nil {
		return
	}

	urls := parser.GetNewTorrentURL()
	for _, url := range urls {
		err := client.AddTorrent(url)
		if err != nil {
			slog.Warn("Failed to add ["+url+"].", "err", err)
		}
	}
}
