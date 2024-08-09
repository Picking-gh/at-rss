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
	"path"
	"sync"

	"github.com/atrox/homedir"
)

const cachePath = "~/.cache/at-rss.gob"

// Cache logs the head for each feed.
type Cache struct {
	path string // cache file path
	mu   sync.RWMutex
	data map[string]string //key:feed url, value: head item guid
}

// NewCache return a new Cache object
func NewCache() (*Cache, error) {
	cache := Cache{}

	path, err := homedir.Expand(cachePath)
	if err != nil {
		slog.Error("Failed to locate cache file path.", "err", err)
		return nil, err
	}
	cache.path = path

	err = readGob(cache.path, &cache.data)
	if err != nil {
		slog.Info("Empty cache")
		cache.data = make(map[string]string)
	}
	return &cache, nil
}

// Get return the value associated with the key or an error if the
// cache doesn't contains the key
func (c *Cache) Get(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	if !ok {
		return "", errors.New("no match found for key " + key)
	}
	return v, nil
}

// Set set in the cache the given value with the given key
func (c *Cache) Set(key string, value string) error {
	c.mu.Lock()
	c.data[key] = value

	return writeGob(c.path, c.data)
}

func writeGob(filePath string, object interface{}) error {
	os.Mkdir(path.Dir(filePath), 0744)
	file, err := os.Create(filePath)
	if err != nil {
		slog.Warn("Failed to write cache to disk. May download duplicate files.", "err", err)
		return err
	}
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(object)
	file.Close()
	return err
}

func readGob(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(object)
	file.Close()
	return err
}
