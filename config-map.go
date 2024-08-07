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

// Config is handling the config parsing
// type Config struct {
// 	Server struct {
// 		Url   string
// 		Token string
// 	}
// 	UpdateInterval uint64 `yaml:"update_interval"`
// 	Feeds          []TorrentParser
// }

type tasks struct {
	a []*aria2cTask
	t []*transmissionTask
}

type aria2cTask struct {
	Server struct {
		Url   string
		Token string
	}
	FetchInterval uint64
	TorrentParser
}
type transmissionTask struct {
	Server struct {
		Host string
		Port uint16
		User string
		Pswd string
	}
	FetchInterval uint64
	TorrentParser
}

var validTags = map[string]struct{}{
	"Title": {}, "Link": {}, "Description": {}, "Enclosure": {}, "GUID": {},
}

// conf file
// feed1:
//     aria2c:
//         url:  "ws://localhost:6800/jsonrpc"
//         token: "abcd"
//     feed: http://example.com/feed1
//     filter:
//         include:
//             - big brother, little brother
//             - brother
//             - sister
//         exclude:
//             - man
//     extracter:
//         tag: link
//         pattern: abcdefg
// feed2:
//     transmission:
//         host:  "localhost"
//         port: 9091
//         username: "admin"
//         password: "12345678"
//     interval: 30
//     feed: http://example.com/feed2
// feed3:
//     transmission:
//     feed: http://example.com/feed3

// NewConfig return a new Config object
func LoadConfig(filename string) (*tasks, error) {
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

	for _, value := range config {
		task, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		// keys must contain aria2c/transmission and feed. filter, extracter and interval are optional.
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

		var tp TorrentParser
		for k, v := range task {
			switch strings.ToLower(k) {
			case "feed":
				url, ok := v.(string)
				if !ok {
					err := errors.New("Feed not valid.")
					slog.Error(err.Error())
					return nil, err
				}
				tp.FeedUrl = url
			case "filter":
				filter, ok := v.(map[string][]string)
				if ok {
					tp.Include = filter["include"]
					tp.Exclude = filter["exclude"]
				}
			case "extracter":
				extract, ok := v.(map[string]string)
				if ok {
					tp.Tag = extract["tag"]
					tp.Pattern = extract["pattern"]
				}
			}
		}

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
