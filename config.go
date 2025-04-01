/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/liuzl/gocc"
	"gopkg.in/yaml.v3"
)

// Config represents the top-level configuration structure
type Config struct {
	Tasks map[string]TaskConfig
}

// UnmarshalYAML implements custom unmarshaling to support key-value task format
func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	var rawMap map[string]any
	if err := unmarshal(&rawMap); err != nil {
		return err
	}

	c.Tasks = make(map[string]TaskConfig)
	for name, value := range rawMap {
		var task TaskConfig
		task.Name = name

		// Convert raw interface{} to YAML bytes for TaskConfig parsing
		taskBytes, err := yaml.Marshal(value)
		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(taskBytes, &task); err != nil {
			return fmt.Errorf("failed to parse task %s: %w", name, err)
		}

		// Perform basic validation during unmarshaling
		if task.Aria2c != nil && task.Transmission != nil {
			return fmt.Errorf("task %s: cannot specify both aria2c and transmission", name)
		}
		if task.Aria2c == nil && task.Transmission == nil {
			return fmt.Errorf("task %s: must specify either aria2c or transmission", name)
		}
		if len(task.Feed.URLs) == 0 {
			return fmt.Errorf("task %s: must specify at least one feed URL", name)
		}

		c.Tasks[name] = task
	}

	return nil
}

// TaskConfig represents a single task configuration
type TaskConfig struct {
	Name         string              `yaml:"name,omitempty"`
	Aria2c       *Aria2cConfig       `yaml:"aria2c,omitempty"`
	Transmission *TransmissionConfig `yaml:"transmission,omitempty"`
	Feed         FeedConfig          `yaml:"feed"`
	Filter       *FilterConfig       `yaml:"filter,omitempty"`
	Extracter    *ExtracterConfig    `yaml:"extracter,omitempty"`
	Interval     int                 `yaml:"interval,omitempty"`
}

// Aria2cConfig represents aria2c RPC configuration
type Aria2cConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token,omitempty"`
}

// TransmissionConfig represents transmission RPC configuration
type TransmissionConfig struct {
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// FeedConfig represents feed URL configuration (supports single or multiple URLs)
type FeedConfig struct {
	URLs []string `yaml:"urls"`
}

// UnmarshalYAML implements custom unmarshaling to support both string and []string
func (f *FeedConfig) UnmarshalYAML(unmarshal func(any) error) error {
	// First try to unmarshal as single string
	var singleURL string
	if err := unmarshal(&singleURL); err == nil {
		f.URLs = []string{singleURL}
		return nil
	}

	// Then try to unmarshal as string slice
	var multiURLs []string
	if err := unmarshal(&multiURLs); err == nil {
		f.URLs = multiURLs
		return nil
	}

	// Finally try the original struct format
	var aux struct {
		URLs []string `yaml:"urls"`
	}
	if err := unmarshal(&aux); err != nil {
		return err
	}
	f.URLs = aux.URLs
	return nil
}

// FilterConfig represents content filter configuration
type FilterConfig struct {
	Include []string `yaml:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty"`
}

// ExtracterConfig represents extraction configuration
type ExtracterConfig struct {
	Tag     string `yaml:"tag"`
	Pattern string `yaml:"pattern"`
}

const (
	defaultAria2cRpcUrl        = "ws://localhost:6800/jsonrpc"
	defaultTransmissionRpcHost = "localhost"
	defaultTransmissionRpcPort = 9091
	defaultFetchInterval       = 10
)

var validTags = map[string]struct{}{
	"title": {}, "link": {}, "description": {}, "enclosure": {}, "guid": {},
}

// LoadConfig loads and validates the configuration from YAML file
func LoadConfig(filename string) ([]*Task, error) {
	config, err := loadYAMLConfig(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cc, err := gocc.New("t2s")
	if err != nil {
		slog.Warn("Failed to initialize Chinese converter", "err", err)
	}

	var tasks []*Task
	for _, taskConfig := range config.Tasks {
		task, err := parseTask(taskConfig, cc)
		if err != nil {
			return nil, fmt.Errorf("invalid task configuration: %w", err)
		}
		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		return nil, errors.New("no valid tasks found in configuration")
	}

	return tasks, nil
}

// loadYAMLConfig reads and unmarshals the YAML configuration file
func loadYAMLConfig(filename string) (*Config, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(source, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &config, nil
}

// parseTask converts TaskConfig to Task
func parseTask(config TaskConfig, cc *gocc.OpenCC) (*Task, error) {
	// Set default interval if not specified
	if config.Interval <= 0 {
		config.Interval = defaultFetchInterval
	}

	task := &Task{
		parserConfig:  &ParserConfig{},
		FeedUrls:      config.Feed.URLs,
		FetchInterval: time.Duration(config.Interval) * time.Minute,
	}

	// Parse RPC configuration
	if config.Aria2c != nil {
		parseAria2cConfig(task, config.Aria2c)
	} else {
		parseTransmissionConfig(task, config.Transmission)
	}

	// Parse filter if specified
	if config.Filter != nil {
		parseFilterConfig(task, config.Filter, cc)
	}

	// Parse extracter if specified
	if config.Extracter != nil {
		if err := parseExtracterConfig(task, config.Extracter); err != nil {
			return nil, err
		}
	}

	return task, nil
}

// parseAria2cConfig processes the aria2c configuration
func parseAria2cConfig(t *Task, cfg *Aria2cConfig) {
	t.ServerConfig.RpcType = "aria2c"
	t.ServerConfig.Url = cfg.URL
	if t.ServerConfig.Url == "" {
		t.ServerConfig.Url = defaultAria2cRpcUrl
	}
	t.ServerConfig.Token = cfg.Token
}

// parseTransmissionConfig processes the transmission configuration
func parseTransmissionConfig(t *Task, cfg *TransmissionConfig) {
	t.ServerConfig.RpcType = "transmission"
	t.ServerConfig.Host = cfg.Host
	if t.ServerConfig.Host == "" {
		t.ServerConfig.Host = defaultTransmissionRpcHost
	}
	t.ServerConfig.Port = cfg.Port
	if t.ServerConfig.Port == 0 {
		t.ServerConfig.Port = defaultTransmissionRpcPort
	}
	t.ServerConfig.Username = cfg.Username
	t.ServerConfig.Password = cfg.Password
}

// parseFilterConfig processes the filter configuration
func parseFilterConfig(t *Task, cfg *FilterConfig, cc *gocc.OpenCC) {
	if cfg == nil {
		return
	}

	t.parserConfig.Include = normalizeAndSimplifyTexts(cc, cfg.Include)
	t.parserConfig.Exclude = normalizeAndSimplifyTexts(cc, cfg.Exclude)
}

// parseExtracterConfig processes and validates the extracter configuration
func parseExtracterConfig(t *Task, cfg *ExtracterConfig) error {
	if cfg == nil {
		return nil
	}

	// Validate tag
	tag := strings.ToLower(cfg.Tag)
	if _, valid := validTags[tag]; !valid {
		return fmt.Errorf("invalid extracter tag: %s", tag)
	}
	t.parserConfig.Tag = tag

	// Validate and compile pattern
	if cfg.Pattern == "" {
		return errors.New("extracter pattern cannot be empty")
	}
	r, err := regexp.Compile(cfg.Pattern)
	if err != nil {
		return fmt.Errorf("invalid extracter pattern: %w", err)
	}
	t.parserConfig.Pattern = cfg.Pattern
	t.parserConfig.r = r
	t.parserConfig.Trick = true

	return nil
}

// normalizeAndSimplifyTexts converts given []string to lowercase and applies Chinese simplification if needed
func normalizeAndSimplifyTexts(cc *gocc.OpenCC, texts []string) []string {
	if cc == nil || len(texts) == 0 {
		return texts
	}

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
