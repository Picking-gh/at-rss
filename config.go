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
	"time"

	"github.com/liuzl/gocc"
	"gopkg.in/yaml.v3"
)

const (
	defaultAria2cRpcUrl        = "ws://localhost:6800/jsonrpc"
	defaultTransmissionRpcHost = "localhost"
	defaultTransmissionRpcPort = 9091
	defaultFetchInterval       = 10
)

var validTags = map[string]struct{}{
	"title": {}, "link": {}, "description": {}, "enclosure": {}, "guid": {},
}

type Tasks []*Task

// LoadConfig returns a Tasks object based on the given filename.
func LoadConfig(filename string) (*Tasks, error) {
	config, err := loadYAMLConfig(filename)
	if err != nil {
		return nil, err
	}

	// The filtering criteria ignore the distinction between traditional and simplified Chinese,
	// so here the Include and Exclude keywords are converted to simplified Chinese.
	cc, err := gocc.New("t2s") // "t2s" traditional Chinese -> simplified Chinese
	if err != nil {
		slog.Warn("Failed to initialize Chinese converter.", "err", err)
	}

	tasks := Tasks{}
	for _, value := range config {
		task, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		taskObj, err := parseTask(task, cc)
		if err != nil {
			slog.Error("Configuration file error.", "err", err)
			return nil, err
		}

		tasks = append(tasks, taskObj)
	}
	return &tasks, nil
}

// loadYAMLConfig reads and unmarshals a YAML configuration file.
func loadYAMLConfig(filename string) (map[string]interface{}, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		slog.Error("Failed to read config file.", "err", err)
		return nil, err
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(source, &config); err != nil {
		slog.Error("Failed to unmarshal config file.", "err", err)
		return nil, err
	}

	return config, nil
}

// parseTask processes each task in the configuration.
func parseTask(task map[string]interface{}, cc *gocc.OpenCC) (*Task, error) {
	_, hasAria2c := task["aria2c"]
	_, hasTransmission := task["transmission"]

	if hasAria2c && hasTransmission {
		return nil, errors.New("both aria2c and transmission RPC servers specified; only one allowed")
	} else if !hasAria2c && !hasTransmission {
		return nil, errors.New("neither aria2c nor transmission RPC server specified")
	}

	if _, hasFeed := task["feed"]; !hasFeed {
		return nil, errors.New("feed section missing")
	}

	t := &Task{pc: &ParserConfig{}, FetchInterval: defaultFetchInterval * time.Minute}

	for k, v := range task {
		switch strings.ToLower(k) {
		case "aria2c":
			parseAria2cConfig(t, v)
		case "transmission":
			parseTransmissionConfig(t, v)
		case "feed":
			if url, urlOk := v.(string); !urlOk {
				return nil, errors.New("feed URL missing")
			} else {
				t.pc.FeedUrl = url
			}
		case "interval":
			t.FetchInterval = time.Duration(getIntOrDefault(v, defaultFetchInterval)) * time.Minute
		case "filter":
			parseFilterConfig(t, v, cc)
		case "extracter":
			if err := parseExtracterConfig(t, v); err != nil {
				return nil, err
			}
		}
	}

	return t, nil
}

// parseAria2cConfig processes the aria2c configuration.
func parseAria2cConfig(t *Task, v interface{}) {
	server, ok := v.(map[string]interface{})
	if !ok || server == nil {
		t.Server.Url = defaultAria2cRpcUrl
	} else {
		t.Server.Url = getStringOrDefault(server["url"], defaultAria2cRpcUrl)
		t.Server.Token, _ = server["token"].(string)
	}
	t.Server.RpcType = "aria2c"
}

// parseTransmissionConfig processes the transmission configuration.
func parseTransmissionConfig(t *Task, v interface{}) {
	server, ok := v.(map[string]interface{})
	if !ok || server == nil {
		t.Server.Host = defaultTransmissionRpcHost
		t.Server.Port = defaultTransmissionRpcPort
	} else {
		t.Server.Host = getStringOrDefault(server["host"], defaultTransmissionRpcHost)
		t.Server.Port = uint16(getIntOrDefault(server["port"], defaultTransmissionRpcPort))
		t.Server.User, _ = server["username"].(string)
		t.Server.Pswd, _ = server["password"].(string)
	}
	t.Server.RpcType = "transmission"
}

// parseFilterConfig processes the filter configuration.
func parseFilterConfig(t *Task, v interface{}, cc *gocc.OpenCC) {
	if tryFilter, ok := v.(map[string]interface{}); ok {
		filter := convertToStringSliceMap(tryFilter)
		t.pc.Include = normalizeAndSimplifyTexts(cc, filter["include"])
		t.pc.Exclude = normalizeAndSimplifyTexts(cc, filter["exclude"])
	}
}

// parseExtracterConfig processes and validates the extracter configuration.
func parseExtracterConfig(t *Task, v interface{}) error {
	extract, ok := v.(map[string]interface{})
	if !ok {
		return errors.New("invalid 'extracter'")
	}

	tag, tagOk := extract["tag"].(string)
	if !tagOk || tag == "" {
		return errors.New("missing 'tag' in extracter")
	}
	tag = strings.ToLower(tag)
	if _, valid := validTags[tag]; !valid {
		return errors.New("invalid 'tag': " + tag + " in extracter")
	}
	t.pc.Tag = tag

	pattern, patternOk := extract["pattern"].(string)
	if !patternOk || pattern == "" {
		return errors.New("missing 'pattern' in extracter")
	}
	r, err := regexp.Compile(pattern)
	if err != nil {
		return errors.New("invalid 'pattern': " + pattern + " in extracter")
	}
	t.pc.Pattern = pattern
	t.pc.r = r

	t.pc.Trick = true

	return nil
}

// normalizeAndSimplifyTexts converts given []string to lowercase and applies Chinese simplification if needed.
func normalizeAndSimplifyTexts(cc *gocc.OpenCC, texts []string) []string {
	if cc == nil {
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

// convertToStringSliceMap converts a map with interface{} values into a map with string slices.
func convertToStringSliceMap(rawMap map[string]interface{}) map[string][]string {
	result := make(map[string][]string)
	for key, value := range rawMap {
		if slice, ok := value.([]interface{}); ok {
			strSlice := make([]string, len(slice))
			i := 0
			for _, item := range slice {
				if str, ok := item.(string); ok {
					strSlice[i] = str
					i++
				}
			}
			result[key] = strSlice
		} else if str, ok := value.(string); ok {
			result[key] = []string{str}
		}
	}
	return result
}

// getStringOrDefault tries to get a string from a interface or returns a default value.
func getStringOrDefault(m interface{}, defaultValue string) string {
	value, ok := m.(string)
	if !ok || value == "" {
		return defaultValue
	}
	return value
}

// getIntOrDefault tries to get an integer from a interface or returns a default value.
func getIntOrDefault(m interface{}, defaultValue int) int {
	if value, ok := m.(int); ok && value > 0 {
		return value
	}
	return defaultValue
}
