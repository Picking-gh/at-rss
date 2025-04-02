/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"slices"
)

const cacheFileName = ".cache/at-rss.json"

// FeedCache holds the items for a specific feed and its last update timestamp.
type FeedCache struct {
	Items     map[string][]string `json:"items"`
	Timestamp time.Time           `json:"timestamp"`
}

// Cache manages the storage and retrieval of RSS feed items.
// The `data` map contains feed URLs as keys, each associated with a FeedCache struct.
// The `filePath` stores the location for saving or loading the cache data.
type Cache struct {
	mu       sync.RWMutex
	data     map[string]FeedCache
	filePath string
}

// NewCache initializes and returns a Cache instance.
func NewCache() (*Cache, error) {
	cache := &Cache{
		data: make(map[string]FeedCache),
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户主目录失败: %w", err)
	}
	cache.filePath = filepath.Join(homeDir, cacheFileName)

	if err := loadCache(cache.filePath, &cache.data); err != nil {
		slog.Warn("缓存加载失败，将初始化空缓存", "err", err)
	}

	return cache, nil
}

// Get returns a copy of non-empty entries from the map associated with the given key
// or an empty map if the key doesn't exist.
// Get returns a copy of non-empty entries from the Items map associated with the given key
// or an empty map if the key doesn't exist.
func (c *Cache) Get(key string) map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if feedCache, exists := c.data[key]; exists {
		result := make(map[string][]string, len(feedCache.Items))
		for k, v := range feedCache.Items {
			// Keep returning even empty slices, as the caller might rely on the key's existence
			result[k] = slices.Clone(v)
		}
		return result
	}
	return make(map[string][]string)
}

// Set stores the provided map under the specified key in the cache and updates the timestamp.
// If 'overwrite' is false, it will only overwrite values for a GUID if the existing slice is empty.
// If 'overwrite' is true, it will always overwrite values for a GUID.
func (c *Cache) Set(key string, value map[string][]string, overwrite bool) {
	if len(value) == 0 {
		return // Nothing to set
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	feedCache, exists := c.data[key]
	if !exists {
		feedCache = FeedCache{
			Items: make(map[string][]string),
		}
	}

	itemsChanged := false
	for k, v := range value {
		existingV, itemExists := feedCache.Items[k]
		shouldSet := overwrite || !itemExists || len(existingV) == 0
		if shouldSet {
			// Only clone if necessary and different
			if !itemExists || !slices.Equal(existingV, v) {
				feedCache.Items[k] = slices.Clone(v) // Store a copy
				itemsChanged = true
			}
		}
	}

	// Update timestamp only if items were actually added or modified
	if itemsChanged || !exists {
		feedCache.Timestamp = time.Now()
		c.data[key] = feedCache // Assign back the potentially modified struct
	}
}

// RemoveNotIn deletes entries from the cache's Items map for a given feed key
// if the entry's key (GUID) is not present in the provided validEntries map.
func (c *Cache) RemoveNotIn(key string, validEntries map[string][]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	feedCache, exists := c.data[key]
	if !exists || len(feedCache.Items) == 0 {
		return // Nothing to remove from
	}

	itemsChanged := false
	for k := range feedCache.Items {
		if _, exists := validEntries[k]; !exists {
			delete(feedCache.Items, k)
			itemsChanged = true
		}
	}

	// Update timestamp if items were removed
	if itemsChanged {
		feedCache.Timestamp = time.Now()
		c.data[key] = feedCache
	}
}

// Flush performs cleanup of old entries and then serializes the cache data
// and writes it to disk at the specified file path.
func (c *Cache) Flush() error {
	c.mu.Lock() // Lock for the entire duration of cleanup and saving
	defer c.mu.Unlock()

	// --- Cleanup Logic ---
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	feedsToDelete := []string{} // Collect keys of feeds to delete entirely

	for feedURL, feedCache := range c.data {
		if feedCache.Timestamp.Before(thirtyDaysAgo) {
			slog.Debug("Checking old feed for cleanup", "url", feedURL, "timestamp", feedCache.Timestamp)
			itemsToDelete := []string{} // Collect keys of items to delete within this feed
			for guid, infoHashes := range feedCache.Items {
				if len(infoHashes) == 0 {
					itemsToDelete = append(itemsToDelete, guid)
				}
			}

			// Delete empty items
			if len(itemsToDelete) > 0 {
				slog.Info("Cleaning up empty items from old feed", "url", feedURL, "count", len(itemsToDelete))
				for _, guid := range itemsToDelete {
					delete(feedCache.Items, guid)
				}
				// Update the map in place (since feedCache is a copy)
				c.data[feedURL] = feedCache
			}

			// Check if the feed itself is now empty
			if len(feedCache.Items) == 0 {
				feedsToDelete = append(feedsToDelete, feedURL)
			}
		}
	}

	// Delete empty feeds
	if len(feedsToDelete) > 0 {
		slog.Info("Cleaning up empty old feeds", "count", len(feedsToDelete), "feeds", feedsToDelete)
		for _, feedURL := range feedsToDelete {
			delete(c.data, feedURL)
		}
	}
	// --- End Cleanup Logic ---

	return saveCache(c.filePath, c.data)
}

// saveCache creates necessary directories and serializes the given object to a file using yaml encoding
// with atomic write operation to prevent data corruption.
func saveCache(filePath string, object interface{}) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0744); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	tmpPath := filePath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpPath)

	enc := json.NewEncoder(file)
	// Use indentation for better readability
	enc.SetIndent("", " ")
	if err := enc.Encode(object); err != nil {
		return fmt.Errorf("JSON编码失败: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("原子重命名失败: %w", err)
	}
	return nil
}

// loadCache opens a file and deserializes its contents into the provided object using yaml encoding.
// Returns nil if file doesn't exist, error for other failures.
func loadCache(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if os.IsNotExist(err) {
		return nil // 文件不存在不算错误
	}
	if err != nil {
		return fmt.Errorf("打开缓存文件失败: %w", err)
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(object)
}
