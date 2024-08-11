/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"encoding/gob"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

const cachePath = ".cache/at-rss.gob"

// Cache stores the head item GUID for each feed URL.
type Cache struct {
	mu   sync.RWMutex
	data map[string]map[string]struct{}
	path string
}

// NewCache creates a new Cache object.
func NewCache() (*Cache, error) {
	cache := &Cache{
		data: make(map[string]map[string]struct{}),
	}

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

// Get retrieves the value associated with the key or returns an error if the key doesn't exist.
func (c *Cache) Get(key string) (map[string]struct{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if value, ok := c.data[key]; ok {
		return value, nil
	}
	return nil, errors.New("no match found for key " + key)
}

// Set stores the given value with the associated key in the cache and persists it.
func (c *Cache) Set(key string, value map[string]struct{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := c.data[key]
	for guid := range value {
		data[guid] = struct{}{}
	}

	return writeGob(c.path, c.data)
}

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

func readGob(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return gob.NewDecoder(file).Decode(object)
}
