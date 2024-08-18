/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"encoding/gob"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const cacheFileName = ".cache/at-rss.gob"

// Cache manages the storage and retrieval of RSS feed items.
// The `data` map contains feed URLs as keys, each associated with a map of GUIDs (Globally Unique Identifiers) and their UNIX timestamps.
// Additionally, there is a special "infoHash" key with a map that tracks the btih (info hash) and its addition time.
// The `filePath` stores the location for saving or loading the cache data.
type Cache struct {
	mu       sync.RWMutex
	data     map[string]map[string]int64 // inner map value is a UNIX timestamp
	filePath string
}

// NewCache initializes and returns a Cache instance.
func NewCache(ctx context.Context) (*Cache, error) {
	cache := &Cache{
		data: make(map[string]map[string]int64),
	}

	// "infoHash" map keeps btih added for 1 day
	cache.data["infoHash"] = make(map[string]int64)
	go cache.startCleanupScheduler(ctx, "infoHash")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Failed to locate user's home directory.", "err", err)
		return nil, err
	}
	cache.filePath = filepath.Join(homeDir, cacheFileName)

	if err := loadCache(cache.filePath, &cache.data); err != nil {
		slog.Warn("Failed to load cache, initializing empty cache.", "err", err)
	}

	return cache, nil
}

// Get returns a copy of the map associated with the given key or an empty map if the key doesn't exist.
func (c *Cache) Get(key string) map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if value, exists := c.data[key]; exists {
		copiedValue := make(map[string]int64)
		for k, v := range value {
			copiedValue[k] = v
		}
		return copiedValue
	}
	return make(map[string]int64)
}

// Set stores the provided map under the specified key in the cache and timestamps the entries.
func (c *Cache) Set(key string, value map[string]int64) {
	if len(value) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data[key]; !exists {
		c.data[key] = make(map[string]int64)
	}
	for k := range value {
		c.data[key][k] = time.Now().Unix()
	}
}

// RemoveNotIn deletes entries from the cache that are not present in the provided map.
// This function operates on the cache map associated with the specified key, usually a feed URL.
func (c *Cache) RemoveNotIn(key string, validEntries map[string]int64) {
	if len(validEntries) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheSubMap := c.data[key]
	for k := range cacheSubMap {
		if _, exists := validEntries[k]; !exists {
			delete(cacheSubMap, k)
		}
	}
}

// Flush serializes the cache data and writes it to disk at the specified file path.
func (c *Cache) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return saveCache(c.filePath, c.data)
}

// cleanupExpiredEntries removes entries from the cache that are older than 24 hours.
func (c *Cache) cleanupExpiredEntries(key string) {
	expirationTime := time.Now().Add(-24 * time.Hour).Unix()

	c.mu.Lock()
	defer c.mu.Unlock()

	if cacheSubMap, exists := c.data[key]; exists {
		for k, timestamp := range cacheSubMap {
			if timestamp < expirationTime {
				delete(cacheSubMap, k)
			}
		}
	}
}

// startCleanupScheduler initiates a cleanup task that runs every hour to remove expired entries.
// The function stops when the context is cancelled.
func (c *Cache) startCleanupScheduler(ctx context.Context, key string) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.cleanupExpiredEntries(key)
		}
	}
}

// saveCache creates necessary directories and serializes the given object to a file using gob encoding.
// It returns an error if directory creation or file writing fails.
func saveCache(filePath string, object interface{}) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0744); err != nil {
		slog.Warn("Failed to create directory for cache file.", "err", err)
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		slog.Warn("Failed to write cache to disk. May download duplicate files.", "err", err)
		return err
	}
	defer file.Close()

	return gob.NewEncoder(file).Encode(object)
}

// loadCache opens a file and deserializes its contents into the provided object using gob encoding.
// Returns an error if the file cannot be opened or if decoding fails.
func loadCache(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return gob.NewDecoder(file).Decode(object)
}
