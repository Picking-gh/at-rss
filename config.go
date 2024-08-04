/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log"
	"os"

	"github.com/liuzl/gocc"
	"gopkg.in/yaml.v3"
)

const defaultServerUrl = "http://localhost:6800/jsonrpc"
const defaultUpdateInterval = 10

// Config is handling the config parsing
type Config struct {
	Server struct {
		Url   string
		Token string
	}
	UpdateInterval uint64 `yaml:"update_interval"`
	Feeds          []Feed
}
type Feed struct {
	Url     string
	Include []string
	Exclude []string
	Trick   bool
	Pattern string
}

// NewConfig return a new Config object
func NewConfig(filename string) *Config {
	var config Config
	source, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(source, &config)
	if err != nil {
		log.Fatal(err)
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
		for _, feed := range config.Feeds {
			feed.Include = convert(cc, feed.Include)
			feed.Exclude = convert(cc, feed.Exclude)
		}
	} else {
		log.Println("Cannot perform traditional and simplified Chinese conversion: ", err)
	}
	return &config
}

// convert convert given []string to the expected type
func convert(cc *gocc.OpenCC, texts []string) []string {
	var simplified []string
	for _, text := range texts {
		result, err := cc.Convert(text)
		if err != nil {
			simplified = append(simplified, text)
		} else {
			simplified = append(simplified, result)
		}
	}
	return simplified
}
