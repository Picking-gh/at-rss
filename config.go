/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"errors"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/liuzl/gocc"
	"gopkg.in/yaml.v3"
)

const defaultServerUrl = "ws://localhost:6800/jsonrpc"
const defaultUpdateInterval = 10

// Config is handling the config parsing
type Config struct {
	Server struct {
		Url   string
		Token string
	}
	UpdateInterval uint64 `yaml:"update_interval"`
	Feeds          []TorrentParser
}

var validTags = map[string]struct{}{
	"Title": {}, "Link": {}, "Description": {}, "Enclosure": {}, "GUID": {},
}

// NewConfig return a new Config object
func NewConfig(filename string) (*Config, error) {
	var config Config
	source, err := os.ReadFile(filename)
	if err != nil {
		slog.Error("Failed to read config file.", "err", err)
		return nil, err
	}
	err = yaml.Unmarshal(source, &config)
	if err != nil {
		slog.Error("Failed to Unmarshal config file.", "err", err)
		return nil, err
	}
	if config.Server.Url == "" {
		config.Server.Url = defaultServerUrl
	}
	if config.UpdateInterval == 0 {
		config.UpdateInterval = defaultUpdateInterval
	}

	// The filtering criteria ignore the distinction between traditional and simplified Chinese,
	// so here the Include and Exclude keywords are converted to simplified Chinese.
	cc, err := gocc.New("t2s") // "t2s" traditional Chinese -> simplified Chinese
	if err == nil {
		for i := range config.Feeds {
			feed := &config.Feeds[i]
			feed.Include = convert(cc, feed.Include)
			feed.Exclude = convert(cc, feed.Exclude)
		}
	} else {
		slog.Warn("Failed to perform traditional and simplified Chinese conversion.", "err", err)
	}

	// If Trick is true, then the tag is validated the pattern is precompiled.
	for i := range config.Feeds {
		if config.Feeds[i].Trick {
			feed := &config.Feeds[i]
			// Validate tag. Transform tag to the same as gofeed.Item fields are except Enclosure. gofeed.Item contains Enclosures
			tag := capitalize(feed.Tag)
			if tag == "Guid" {
				tag = "GUID"
			}
			if _, hasTag := validTags[tag]; !hasTag {
				err := errors.New("Tag [" + feed.Tag + "] invalid. Supported tags are title, link, description, enclosure, and guid.")
				slog.Error(err.Error())
				return nil, err
			}
			feed.Tag = tag
			// Compile pattern
			r, err := regexp.Compile(feed.Pattern)
			if err != nil {
				slog.Error("Pattern [" + feed.Pattern + "] invalid.")
				return nil, err
			}
			feed.r = r
		}
	}

	return &config, nil
}

// convert converts given []string to the expected type, and return in lower case.
func convert(cc *gocc.OpenCC, texts []string) []string {
	var simplified []string
	for _, text := range texts {
		text = strings.TrimSpace(strings.ToLower(text))
		result, err := cc.Convert(text)
		if err != nil {
			simplified = append(simplified, text)
		} else {
			simplified = append(simplified, result)
		}
	}
	return simplified
}

// capitalize turns s to its captitalized form.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}

	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
