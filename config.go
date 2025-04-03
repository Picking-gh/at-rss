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
	"time"

	"github.com/liuzl/gocc"
	"gopkg.in/yaml.v3"
)

// DownloaderConfig represents the downloader configuration within the YAML file.
type DownloaderConfig struct {
	Type string `yaml:"type"` // "aria2c" or "transmission"

	// Aria2c specific fields
	URL   string `yaml:"url,omitempty"`
	Token string `yaml:"token,omitempty"`

	// Transmission specific fields
	Host     string `yaml:"host,omitempty"`
	Port     uint16 `yaml:"port,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// TaskConfig represents a single task configuration.
type TaskConfig struct {
	Name        string             `yaml:"-"` // Name is derived from the map key, not parsed from YAML directly here.
	Downloaders []DownloaderConfig `yaml:"downloaders"`
	Feed        FeedConfig         `yaml:"feed"`
	Filter      *FilterConfig      `yaml:"filter,omitempty"`
	Extracter   *ExtracterConfig   `yaml:"extracter,omitempty"`
	Interval    int                `yaml:"interval,omitempty"`
}

// Removed obsolete Aria2cConfig and TransmissionConfig definitions

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
	taskConfigs, err := loadYAMLConfig(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate basic requirements for each task after successful YAML parsing
	if len(taskConfigs) == 0 {
		// Handle case where the YAML file is empty or contains no valid tasks
		return nil, errors.New("no tasks defined in configuration")
	}
	for name, taskConfig := range taskConfigs {
		if len(taskConfig.Downloaders) == 0 {
			return nil, fmt.Errorf("task %q: must specify at least one downloader", name)
		}
		if len(taskConfig.Feed.URLs) == 0 {
			return nil, fmt.Errorf("task %q: must specify at least one feed URL", name)
		}
		// Task name ('name') is available here if needed for future validation logic
		// before calling parseTask.
	}

	cc, err := gocc.New("t2s") // Initialize Chinese converter
	if err != nil {
		slog.Warn("Failed to initialize Chinese converter", "err", err)
	}

	var tasks []*Task
	for name, taskConfig := range taskConfigs {
		// Pass name to parseTask for better error context
		task, err := parseTask(name, taskConfig, cc)
		if err != nil {
			// Add task name to the wrapper message
			return nil, fmt.Errorf("invalid configuration for task %q: %w", name, err)
		}
		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		// This error means tasks were defined in YAML, but none were successfully parsed into Task objects.
		return nil, errors.New("no valid tasks could be parsed from the configuration")
	}

	return tasks, nil
}

// loadYAMLConfig reads and unmarshals the YAML configuration file
func loadYAMLConfig(filename string) (map[string]TaskConfig, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var taskConfigs map[string]TaskConfig
	if err := yaml.Unmarshal(source, &taskConfigs); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return taskConfigs, nil
}

// parseTask converts TaskConfig to Task, accepting the task name for context
func parseTask(name string, config TaskConfig, cc *gocc.OpenCC) (*Task, error) {
	// Set default interval if not specified
	if config.Interval <= 0 {
		config.Interval = defaultFetchInterval
	}

	task := &Task{
		parserConfig:  &ParserConfig{},
		FeedUrls:      config.Feed.URLs,
		FetchInterval: time.Duration(config.Interval) * time.Minute,
		Downloaders:   make([]ParsedDownloaderConfig, 0, len(config.Downloaders)),
	}

	// Parse downloader configurations from YAML struct to internal struct
	for i, dlYAML := range config.Downloaders {
		dlConfig, err := parseDownloaderConfig(dlYAML)
		if err != nil {
			// Use the passed 'name' instead of config.Name (which isn't set)
			return nil, fmt.Errorf("invalid downloader config at index %d for task %q: %w", i, name, err)
		}
		task.Downloaders = append(task.Downloaders, dlConfig)
	}

	// Parse filter if specified
	if config.Filter != nil {
		parseFilterConfig(task, config.Filter, cc)
	}

	// Parse extracter if specified
	if config.Extracter != nil {
		if err := parseExtracterConfig(task, config.Extracter); err != nil {
			// Add task name context here too
			return nil, fmt.Errorf("invalid extracter config for task %q: %w", name, err)
		}
	}

	return task, nil
}

// parseDownloaderConfig converts the YAML DownloaderConfig representation
// to the internal ParsedDownloaderConfig struct used by tasks.
func parseDownloaderConfig(dlYAML DownloaderConfig) (ParsedDownloaderConfig, error) {
	// Create the internal ParsedDownloaderConfig struct (defined in task.go)
	cfg := ParsedDownloaderConfig{
		RpcType: strings.ToLower(dlYAML.Type),
	}

	switch cfg.RpcType {
	case "aria2c":
		cfg.Url = dlYAML.URL
		if cfg.Url == "" {
			cfg.Url = defaultAria2cRpcUrl
		}
		cfg.Token = dlYAML.Token
	case "transmission":
		cfg.Host = dlYAML.Host
		if cfg.Host == "" {
			cfg.Host = defaultTransmissionRpcHost
		}
		cfg.Port = dlYAML.Port
		if cfg.Port == 0 {
			cfg.Port = defaultTransmissionRpcPort
		}
		cfg.Username = dlYAML.Username
		cfg.Password = dlYAML.Password
	default:
		// Return zero value of ParsedDownloaderConfig on error
		return ParsedDownloaderConfig{}, fmt.Errorf("unknown downloader type: %s", dlYAML.Type)
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

	// Validate tag
	tag := strings.ToLower(cfg.Tag)
	if _, valid := validTags[tag]; !valid {
		return fmt.Errorf("invalid extracter tag: %s", tag)
	}

	// Validate pattern
	if cfg.Pattern == "" {
		return errors.New("extracter pattern cannot be empty")
	}

	// Create new parser config
	pc, err := NewParserConfig(nil, nil, true, cfg.Pattern, tag)
	if err != nil {
		return fmt.Errorf("invalid extracter configuration: %w", err)
	}

	// Replace the parser config
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
