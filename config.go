/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log"
	"os"

	"github.com/go-yaml/yaml"
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
	Feeds          []string
	Keywords       []string
	Trick          bool
	Pattern        string
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
	return &config
}
