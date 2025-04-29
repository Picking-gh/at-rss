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
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/liuzl/gocc"
)

// DownloaderConfig represents the downloader configuration within the YAML file.
type DownloaderConfig struct {
	Type     string `yaml:"type" json:"type"` // "aria2c" or "transmission"
	Host     string `yaml:"host,omitempty" json:"host,omitempty"`
	Port     uint16 `yaml:"port,omitempty" json:"port,omitempty"`
	RpcPath  string `yaml:"rpcPath,omitempty" json:"rpcPath,omitempty"`   // RPC path (e.g., "/jsonrpc", "/transmission/rpc")
	UseHttps bool   `yaml:"useHttps,omitempty" json:"useHttps,omitempty"` // Use HTTPS instead of HTTP

	// Authentication
	Token    string `yaml:"token,omitempty" json:"token,omitempty"`       // For aria2c
	Username string `yaml:"username,omitempty" json:"username,omitempty"` // For transmission
	Password string `yaml:"password,omitempty" json:"password,omitempty"` // For transmission

	AutoCleanUp bool `yaml:"autoCleanUp,omitempty" json:"autoCleanUp,omitempty"` // Option to automatically clean up completed tasks
}

// TaskConfig represents a single task configuration.
type TaskConfig struct {
	Name        string             `yaml:"-" json:"-"` // Name is derived from the map key, not parsed from YAML directly here.
	Downloaders []DownloaderConfig `yaml:"downloaders" json:"downloaders"`
	Feeds       FeedsConfig        `yaml:"feeds" json:"feeds"`
	Filter      *FilterConfig      `yaml:"filter,omitempty" json:"filter,omitempty"`
	Extracter   *ExtracterConfig   `yaml:"extracter,omitempty" json:"extracter,omitempty"`
	Interval    int                `yaml:"interval,omitempty" json:"interval,omitempty"`
}

// FeedsConfig represents feed configuration (supports single string or string array)
type FeedsConfig []string

// UnmarshalYAML implements custom unmarshaling to support both string and []string
func (f *FeedsConfig) UnmarshalYAML(unmarshal func(any) error) error {
	// First try to unmarshal as single string
	var singleURL string
	if err := unmarshal(&singleURL); err == nil {
		*f = []string{singleURL}
		return nil
	}

	// Then try to unmarshal as string slice
	var multiURLs []string
	if err := unmarshal(&multiURLs); err == nil {
		*f = multiURLs
		return nil
	}

	return errors.New("feeds must be a string or string array")
}

// FilterConfig represents content filter configuration
type FilterConfig struct {
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// ExtracterConfig represents extraction configuration
type ExtracterConfig struct {
	Tag     string `yaml:"tag" json:"tag"`
	Pattern string `yaml:"pattern" json:"pattern"`
}

const (
	// Default values
	defaultAria2cHost          = "localhost"
	defaultAria2cPort          = 6800
	defaultAria2cRpcPath       = "/jsonrpc"
	defaultTransmissionHost    = "localhost"
	defaultTransmissionPort    = 9091
	defaultTransmissionRpcPath = "/transmission/rpc"
	defaultFetchInterval       = 10
	defaultUseHttps            = false
)

var validTags = map[string]struct{}{
	"title": {}, "link": {}, "description": {}, "enclosure": {}, "guid": {},
}

var (
	// configLock protects access to the config file.
	// Consider potential race conditions if main.go reloads config while API is writing.
	configLock sync.RWMutex
)

// LoadConfig loads and validates the configuration from YAML file
func LoadConfig(filename string, fetchInterval int) ([]*Task, error) {
	taskConfigs, err := LoadYAMLConfig(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate basic requirements for each task after successful YAML parsing
	if len(taskConfigs) == 0 {
		// return nil, errors.New("no tasks defined in configuration")
		return nil, nil
	}
	for name, taskConfig := range taskConfigs {
		if len(taskConfig.Downloaders) == 0 {
			return nil, fmt.Errorf("task %q: must specify at least one downloader", name)
		}
		if len(taskConfig.Feeds) == 0 {
			return nil, fmt.Errorf("task %q: must specify at least one feed", name)
		}
	}

	cc, err := gocc.New("t2s") // Initialize Chinese converter
	if err != nil {
		slog.Warn("Failed to initialize Chinese converter", "err", err)
	}

	var tasks []*Task
	for name, taskConfig := range taskConfigs {
		task, err := parseTask(name, taskConfig, cc, fetchInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid configuration for task %q: %w", name, err)
		}
		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		return nil, errors.New("no valid tasks could be parsed from the configuration")
	}

	return tasks, nil
}

// loadYAMLConfig reads and unmarshals the YAML configuration file
func LoadYAMLConfig(cfgPath string) (map[string]TaskConfig, error) {
	configLock.Lock()
	defer configLock.Unlock()

	source, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var taskConfigs map[string]TaskConfig
	if err := yaml.Unmarshal(source, &taskConfigs); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return taskConfigs, nil
}

// SaveYAMLConfig saves the task configurations back to the YAML file.
func SaveYAMLConfig(cfgPath string, taskConfigs map[string]TaskConfig) error {
	configLock.Lock()
	defer configLock.Unlock()

	data, err := yaml.Marshal(taskConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Use 0600 for potentially sensitive config data
	err = os.WriteFile(cfgPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write to config file %s: %w", cfgPath, err)
	}

	slog.Info("Configuration saved successfully via API", "path", cfgPath)
	return nil
}

// parseTask converts TaskConfig to Task, accepting the task name for context
func parseTask(name string, config TaskConfig, cc *gocc.OpenCC, fetchInterval int) (*Task, error) {
	if config.Interval <= 0 {
		if fetchInterval > 0 {
			config.Interval = fetchInterval
		} else {
			config.Interval = defaultFetchInterval
		}
	}

	task := &Task{
		Name:          name,
		parserConfig:  &ParserConfig{},
		FeedUrls:      config.Feeds,
		FetchInterval: time.Duration(config.Interval) * time.Minute,
		Downloaders:   make([]ParsedDownloaderConfig, 0, len(config.Downloaders)),
	}

	for i, dlYAML := range config.Downloaders {
		dlConfig, err := parseDownloaderConfig(dlYAML)
		if err != nil {
			return nil, fmt.Errorf("invalid downloader config at index %d for task %q: %w", i, name, err)
		}
		task.Downloaders = append(task.Downloaders, dlConfig)
	}

	if config.Filter != nil {
		parseFilterConfig(task, config.Filter, cc)
	}

	if config.Extracter != nil {
		if err := parseExtracterConfig(task, config.Extracter); err != nil {
			return nil, fmt.Errorf("invalid extracter config for task %q: %w", name, err)
		}
	}

	return task, nil
}

// parseDownloaderConfig converts the YAML DownloaderConfig representation
// to the internal ParsedDownloaderConfig struct used by tasks.
func parseDownloaderConfig(dlYAML DownloaderConfig) (ParsedDownloaderConfig, error) {
	rpcType := strings.ToLower(dlYAML.Type)
	if rpcType != "aria2c" && rpcType != "transmission" {
		return ParsedDownloaderConfig{}, fmt.Errorf("unknown downloader type: %s", dlYAML.Type)
	}

	// Set defaults based on type
	host := dlYAML.Host
	port := dlYAML.Port
	rpcPath := dlYAML.RpcPath
	useHttps := dlYAML.UseHttps

	if host == "" {
		if rpcType == "aria2c" {
			host = defaultAria2cHost
		} else {
			host = defaultTransmissionHost
		}
	}
	if port == 0 {
		if rpcType == "aria2c" {
			port = defaultAria2cPort
		} else {
			port = defaultTransmissionPort
		}
	}
	if rpcPath == "" {
		if rpcType == "aria2c" {
			rpcPath = defaultAria2cRpcPath
		} else {
			rpcPath = defaultTransmissionRpcPath
		}
	}
	// Ensure rpcPath starts with a slash
	if !strings.HasPrefix(rpcPath, "/") {
		rpcPath = "/" + rpcPath
	}

	// Build URL
	scheme := "http"
	if useHttps {
		scheme = "https"
	}
	rpcUrl := fmt.Sprintf("%s://%s:%d%s", scheme, host, port, rpcPath)

	// Create the internal ParsedDownloaderConfig struct (defined in task.go)
	cfg := ParsedDownloaderConfig{
		RpcType:     rpcType,
		RpcUrl:      rpcUrl, // Store the constructed URL
		AutoCleanUp: dlYAML.AutoCleanUp,
	}

	// Handle authentication
	if rpcType == "aria2c" {
		cfg.Token = dlYAML.Token // Token can be empty
	} else { // transmission
		cfg.Username = dlYAML.Username // Username can be empty
		cfg.Password = dlYAML.Password // Password can be empty
	}

	return cfg, nil
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

	tag := strings.ToLower(cfg.Tag)
	if _, valid := validTags[tag]; !valid {
		return fmt.Errorf("invalid extracter tag: %s", tag)
	}

	if cfg.Pattern == "" {
		return errors.New("extracter pattern cannot be empty")
	}

	pc, err := NewParserConfig(nil, nil, true, cfg.Pattern, tag)
	if err != nil {
		return fmt.Errorf("invalid extracter configuration: %w", err)
	}

	t.parserConfig = pc
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
