/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"slices"

	"gopkg.in/yaml.v3"
)

const cacheFileName = ".cache/at-rss.yml"

// Cache manages the storage and retrieval of RSS feed items.
// The `data` map contains feed URLs as keys, each associated with a map of GUIDs (Globally Unique Identifiers) and their torrent infoHashes if added to rpc client.
// The `filePath` stores the location for saving or loading the cache data.
type Cache struct {
	mu       sync.RWMutex
	data     map[string]map[string][]string // inner map value is a slice of added torrent infoHashes
	filePath string
}

// NewCache initializes and returns a Cache instance.
func NewCache() (*Cache, error) {
	cache := &Cache{
		data: make(map[string]map[string][]string),
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
func (c *Cache) Get(key string) map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if value, exists := c.data[key]; exists {
		result := make(map[string][]string, len(value))
		for k, v := range value {
			if len(v) > 0 {
				result[k] = slices.Clone(v)
			}
		}
		return result
	}
	return make(map[string][]string)
}

// Set stores the provided map under the specified key in the cache.
// If 'overwrite' is false, it will only overwrite values when the existing slice is empty.
// If 'overwrite' is true, it will always overwrite values.
func (c *Cache) Set(key string, value map[string][]string, overwrite bool) {
	if len(value) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data[key]; !exists {
		c.data[key] = make(map[string][]string)
	}
	for k, v := range value {
		if overwrite {
			c.data[key][k] = v
		} else {
			if len(c.data[key][k]) == 0 {
				c.data[key][k] = v
			}
		}
	}
}

// RemoveNotIn deletes entries from the cache that are not present in the provided map.
// This function operates on the cache map associated with the specified key, usually a feed URL.
func (c *Cache) RemoveNotIn(key string, validEntries map[string][]string) {
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

	enc := yaml.NewEncoder(file)
	if err := enc.Encode(object); err != nil {
		return fmt.Errorf("YAML编码失败: %w", err)
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

	return yaml.NewDecoder(file).Decode(object)
}
