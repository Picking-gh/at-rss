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

const defaultAria2cRpcUrl = "ws://localhost:6800/jsonrpc"
const defaultTransmissionRpcHost = "localhost"
const defaultTransmissionRpcPort = 9091
const defaultFetchInterval = 10

var validTags = map[string]struct{}{
	"Title": {}, "Link": {}, "Description": {}, "Enclosure": {}, "GUID": {},
}

type Tasks []*Task

// LoadConfig return a Tasks object based on filename.
func LoadConfig(filename string) (*Tasks, error) {
	var config map[string]interface{}
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

	// The filtering criteria ignore the distinction between traditional and simplified Chinese,
	// so here the Include and Exclude keywords are converted to simplified Chinese.
	cc, err := gocc.New("t2s") // "t2s" traditional Chinese -> simplified Chinese
	if err != nil {
		slog.Warn("Failed to perform traditional and simplified Chinese conversion.", "err", err)
	}

	ts := Tasks{}
	for _, value := range config {
		task, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		// keys must contain aria2c/transmission and feed. filter, extracter and interval are optional. All in lowercase.
		_, hasAria2c := task["aria2c"]
		_, hasTrasmission := task["transmission"]
		if hasAria2c && hasTrasmission {
			err := errors.New("Accept one rpc server but two exit.")
			slog.Error(err.Error())
			return nil, err
		} else if !hasAria2c && !hasTrasmission {
			err := errors.New("Need one rpc server but none exits.")
			slog.Error(err.Error())
			return nil, err
		}
		_, hasFeed := task["feed"]
		if !hasFeed {
			err := errors.New("Need feed but none exits.")
			slog.Error(err.Error())
			return nil, err
		}

		t := &Task{tp: &TorrentParser{}}
		for k, v := range task {
			switch strings.ToLower(k) {
			case "aria2c":
				server, ok := v.(map[string]interface{})
				if ok && server != nil {
					t.Server.Url, ok = server["url"].(string)
					if !ok || len(t.Server.Url) == 0 {
						t.Server.Url = defaultAria2cRpcUrl
					}
					t.Server.Token, _ = server["token"].(string)
				} else {
					t.Server.Url = defaultAria2cRpcUrl
				}
				t.Server.RpcType = "aria2c"
			case "transmission":
				server, ok := v.(map[string]interface{})
				if ok && server != nil {
					t.Server.Host, ok = server["host"].(string)
					if !ok || len(t.Server.Host) == 0 {
						t.Server.Host = defaultTransmissionRpcHost
					}
					port, ok := server["port"].(int)
					if !ok || port <= 0 {
						t.Server.Port = defaultTransmissionRpcPort
					} else {
						t.Server.Port = (uint16(port))
					}
					t.Server.User, _ = server["username"].(string)
					t.Server.Pswd, _ = server["password"].(string)
				} else {
					t.Server.Host = defaultTransmissionRpcHost
					t.Server.Port = defaultTransmissionRpcPort
				}
				t.Server.RpcType = "transmission"
			case "feed":
				url, ok := v.(string)
				if !ok || len(url) == 0 {
					err := errors.New("Feed not valid.")
					slog.Error(err.Error())
					return nil, err
				}
				t.tp.FeedUrl = url
			case "interval":
				interval, _ := v.(int)
				if interval <= 0 {
					interval = defaultFetchInterval
				}
				t.FetchInterval = uint64(interval)
			case "filter":
				filter, ok := v.(map[string][]string)
				if ok {
					t.tp.Include = convert(cc, filter["include"])
					t.tp.Exclude = convert(cc, filter["exclude"])
				}
			case "extracter":
				extract, ok := v.(map[string]string)
				if ok && extract != nil {
					// Validate tag. Transform tag to the same as gofeed.Item fields are except Enclosure. gofeed.Item contains Enclosures
					tag := capitalize(extract["tag"])
					if tag == "Guid" {
						tag = "GUID"
					}
					if _, hasTag := validTags[tag]; !hasTag {
						err := errors.New("Tag [" + tag + "] invalid. Supported tags are title, link, description, enclosure, and guid.")
						slog.Error(err.Error())
						return nil, err
					}
					t.tp.Tag = tag
					// Compile pattern
					pattern := extract["pattern"]
					r, err := regexp.Compile(pattern)
					if err != nil {
						slog.Error("Pattern [" + pattern + "] invalid.")
						return nil, err
					}
					t.tp.r = r
					// Trick is true, only if tag is validated pattern is precompiled.
					t.tp.Trick = true
				}
			}
		}
		ts = append(ts, t)
	}
	return &ts, nil
}

// convert converts given []string to the expected type, and return in lower case.
func convert(cc *gocc.OpenCC, texts []string) []string {
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

// capitalize turns s to its captitalized form.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}

	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
