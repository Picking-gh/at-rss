/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"encoding/gob"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const cachePath = ".cache/at-rss.gob"

// Cache is a struct that holds information related to RSS feed items.
// The `data` field is a map where each key is a feed URL (string), and the value
// is another map that stores the GUIDs (Globally Unique Identifiers) of all the
// items in that particular feed. Additionally, this map includes an "infoHash" key,
// whose value is a map where the key is the btih (info hash) and the value is the
// Unix timestamp indicating when the btih was added. The `path` field is a string
// that specifies the file path where the cache data may be stored or retrieved from.
type Cache struct {
	mu   sync.RWMutex
	data map[string]map[string]int64 //inner map value is UNIX tiamstamp
	path string
}

// NewCache creates a new Cache object.
func NewCache(ctx context.Context) (*Cache, error) {
	cache := &Cache{
		data: make(map[string]map[string]int64),
	}
	cache.data["infoHash"] = make(map[string]int64)
	go cache.startCleanupScheduler(ctx, "infoHash")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Failed to locate user's home directory.", "err", err)
		return nil, err
	}
	cache.path = filepath.Join(homeDir, cachePath)

	if err := readGob(cache.path, &cache.data); err != nil {
		slog.Warn("Failed to read cache, initializing empty cache.", "err", err)
	}

	return cache, nil
}

// Get returns a copy of the value associated with the key or returns an error if the key doesn't exist.
func (c *Cache) Get(key string) (map[string]int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if value, ok := c.data[key]; ok {
		dest := make(map[string]int64)
		for k, v := range value {
			dest[k] = v
		}
		return dest, nil
	}
	return nil, errors.New("no match found for key " + key)
}

// Set stores the given value with the associated key in the cache and persists it.
func (c *Cache) Set(key string, value map[string]int64) {
	if len(value) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data[key]; !exists {
		c.data[key] = make(map[string]int64)
	}
	data := c.data[key]
	for key2 := range value {
		data[key2] = time.Now().Unix()
	}
}

// RemoveNotIn removes entries from the cache that are not present in the provided value map.
// It operates on the cache data corresponding to the given key. If the value map is empty, the function returns immediately.
// The function locks the cache for thread-safe access, retrieves the map associated with the specified key,
// and deletes any entries from that map which are not present in the provided value map.
//
// Parameters:
//   - key: The key used to access the cache map for a specific feed URL.
//   - value: A map where keys represent the valid entries. Any entry in the cache map with a key not present in this map will be removed.
func (c *Cache) RemoveNotIn(key string, value map[string]int64) {
	if len(value) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheItems := c.data[key]
	// Remove cache items that are not in the provided value map
	for key2 := range cacheItems {
		if _, found := value[key2]; !found {
			delete(cacheItems, key2)
		}
	}
}

// Flush serializes the cache data and writes it to disk at the specified path.
// It locks the cache for thread-safe operations and uses the writeGob function
// to perform the actual writing. It returns an error if the write operation fails.
func (c *Cache) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return writeGob(c.path, c.data)
}

// cleanupExpiredEntries removes entries from the cache that have timestamps older than 24 hours.
// It operates on the cache data corresponding to the specified key. If the key does not exist in the cache,
// the function returns immediately. It locks the cache for thread-safe access, retrieves the map associated
// with the specified key, and deletes entries that are older than one day.
func (c *Cache) cleanupExpiredEntries(key string) {
	oneDayAgo := time.Now().Add(-24 * time.Hour).Unix()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Access the cache for the specific key
	subMap, exists := c.data[key]
	if !exists {
		return
	}

	// Iterate and delete expired entries
	for key, timestamp := range subMap {
		if timestamp < oneDayAgo {
			delete(subMap, key)
		}
	}
}

// startCleanupScheduler initiates a scheduled cleanup task that runs every hour.
// It takes a context for cancellation and a key to specify which cache map to clean.
// The function uses a ticker to trigger the cleanup function every hour. It listens for context cancellation
// and stops the cleanup task when the context is cancelled.
func (c *Cache) startCleanupScheduler(ctx context.Context, key string) {
	ticker := time.NewTicker(time.Hour) // Trigger cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Exit when context is cancelled
			return
		case <-ticker.C:
			c.cleanupExpiredEntries(key)
		}
	}
}

// writeGob creates the necessary directories and serializes the given object
// to a file using the gob encoding format. It ensures that the directory structure
// exists before attempting to create the file. Returns an error if directory creation
// or file writing fails.
func writeGob(filePath string, object interface{}) error {
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

// readGob opens a file and deserializes its contents into the provided object
// using the gob encoding format. Returns an error if the file cannot be opened
// or if decoding fails.
func readGob(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return gob.NewDecoder(file).Decode(object)
}
