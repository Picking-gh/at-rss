package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
)

func TestConfigParsing(t *testing.T) {
	tests := []struct {
		name    string
		yamlStr string
		wantErr bool
		errMsg  string
	}{
		{
			name: "single feed with single aria2c downloader",
			yamlStr: `
feed1:
  downloaders:
    - type: aria2c
      url: "ws://localhost:6800/jsonrpc"
      token: "abcd"
  feed: "http://example.com/feed1"
  interval: 30`,
			wantErr: false,
		},
		{
			name: "multi-line feed with single transmission downloader",
			yamlStr: `
feed2:
  downloaders:
    - type: transmission
      host: "localhost"
      port: 9091
  feed:
    - http://example.com/feed1
    - http://example.com/feed2`,
			wantErr: false,
		},
		{
			name: "single downloader with filter and extracter",
			yamlStr: `
feed3:
  downloaders:
    - type: aria2c
      url: "ws://localhost:6800/jsonrpc"
  feed: "http://example.com/feed3"
  filter:
    include:
      - "keyword1,keyword2"
      - "keyword3"
    exclude:
      - "badword1"
  extracter:
    tag: "link"
    pattern: "[0-9a-f]{40}"`,
			wantErr: false,
		},
		{
			name: "single downloader using defaults",
			yamlStr: `
feed4:
  downloaders:
    - type: aria2c # URL will default
  feed: "http://example.com/feed4"`,
			wantErr: false,
		},
		{
			name: "multiple downloaders (aria2c and transmission)",
			yamlStr: `
feed5:
  downloaders:
    - type: aria2c
      token: "abc"
    - type: transmission
      host: "nas.local"
  feed: "http://example.com/feed5"`,
			wantErr: false,
		},
		{
			name: "multiple downloaders of same type",
			yamlStr: `
feed6:
  downloaders:
    - type: aria2c
      url: "ws://localhost:6800/jsonrpc"
    - type: aria2c
      url: "ws://remote:6800/jsonrpc"
      token: "def"
  feed: "http://example.com/feed6"`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var taskConfigs map[string]TaskConfig
			err := yaml.Unmarshal([]byte(tt.yamlStr), &taskConfigs)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(taskConfigs) == 0 {
				t.Error("Expected tasks but got none")
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Helper to build default URLs for tests
	defaultAria2cHttpUrl := "http://" + defaultAria2cHost + ":" + fmt.Sprintf("%d", defaultAria2cPort) + defaultAria2cRpcPath
	defaultTransmissionHttpUrl := "http://" + defaultTransmissionHost + ":" + fmt.Sprintf("%d", defaultTransmissionPort) + defaultTransmissionRpcPath

	type expectedTask struct {
		FeedURLCount          int
		DownloaderCount       int
		FirstDownloaderType   string
		FirstDownloaderRpcUrl string // Changed from URL/Host
		FetchIntervalMinutes  int
	}

	tests := []struct {
		name         string
		yamlContent  string
		wantTasks    int
		expectedData []expectedTask
	}{
		{
			name: "single task, single aria2c downloader",
			yamlContent: `
feed1:
  downloaders:
    - type: aria2c
      host: "custom.aria2c.host" # Custom host, default port/path/http
  feed: "http://example.com/feed1"`,
			wantTasks: 1,
			expectedData: []expectedTask{
				// Note: ws:// is no longer directly supported in config, it defaults to http/https
				{FeedURLCount: 1, DownloaderCount: 1, FirstDownloaderType: "aria2c", FirstDownloaderRpcUrl: "http://custom.aria2c.host:6800/jsonrpc", FetchIntervalMinutes: defaultFetchInterval},
			},
		},
		{
			name: "single task, multiple downloaders (aria2c default, transmission custom)",
			yamlContent: `
feed2:
  downloaders:
    - type: aria2c # Uses default URL
    - type: transmission
      host: "nas.local" # Custom host, default port
  feed: ["http://example.com/feed2a", "http://example.com/feed2b"] # Multiple feeds
  interval: 20 # Custom interval`,
			wantTasks: 1,
			expectedData: []expectedTask{
				{FeedURLCount: 2, DownloaderCount: 2, FirstDownloaderType: "aria2c", FirstDownloaderRpcUrl: defaultAria2cHttpUrl, FetchIntervalMinutes: 20},
			},
		},
		{
			name: "multiple tasks with different configs",
			yamlContent: `
task_a: # Uses defaults
  downloaders: [{type: aria2c}]
  feed: "http://a.com"
task_b: # Custom interval and downloader
  downloaders: [{type: transmission, host: "192.168.1.1", port: 9091}]
  feed: "http://b.com"
  interval: 5`,
			wantTasks: 2,
			expectedData: []expectedTask{
				{FeedURLCount: 1, DownloaderCount: 1, FirstDownloaderType: "aria2c", FirstDownloaderRpcUrl: defaultAria2cHttpUrl, FetchIntervalMinutes: defaultFetchInterval},
				{FeedURLCount: 1, DownloaderCount: 1, FirstDownloaderType: "transmission", FirstDownloaderRpcUrl: "http://192.168.1.1:9091/transmission/rpc", FetchIntervalMinutes: 5},
			},
		},
		{
			name: "single task, transmission downloader using defaults",
			yamlContent: `
feed_tm_defaults:
  downloaders:
    - type: transmission # Uses default host/port
  feed: "http://example.com/tm_defaults"`,
			wantTasks: 1,
			expectedData: []expectedTask{
				{FeedURLCount: 1, DownloaderCount: 1, FirstDownloaderType: "transmission", FirstDownloaderRpcUrl: defaultTransmissionHttpUrl, FetchIntervalMinutes: defaultFetchInterval},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp(".", "test-config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			filePath := tmpFile.Name()
			defer os.Remove(filePath)

			if _, err := tmpFile.WriteString(tt.yamlContent); err != nil {
				tmpFile.Close()
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			if err := tmpFile.Close(); err != nil {
				t.Fatalf("Failed to close temp file: %v", err)
			}

			tasks, err := LoadConfig(filePath)

			if err != nil {
				t.Fatalf("LoadConfig() returned unexpected error for test case '%s': %v", tt.name, err)
			}

			if len(tasks) != tt.wantTasks {
				t.Fatalf("LoadConfig() got %d tasks, want %d for test case '%s'", len(tasks), tt.wantTasks, tt.name)
			}

			if len(tt.expectedData) > 0 {
				if len(tasks) < len(tt.expectedData) {
					t.Fatalf("LoadConfig() parsed %d tasks, but expected data for %d tasks for test case '%s'", len(tasks), len(tt.expectedData), tt.name)
				}
				for i, expected := range tt.expectedData {
					task := tasks[i]
					if len(task.FeedUrls) != expected.FeedURLCount {
						t.Errorf("Task %d: got %d feed URLs, want %d", i, len(task.FeedUrls), expected.FeedURLCount)
					}
					if len(task.Downloaders) != expected.DownloaderCount {
						t.Errorf("Task %d: got %d downloaders, want %d", i, len(task.Downloaders), expected.DownloaderCount)
					}
					if len(task.Downloaders) > 0 {
						firstDownloader := task.Downloaders[0]
						if firstDownloader.RpcType != expected.FirstDownloaderType {
							t.Errorf("Task %d, Downloader 0: got type %q, want %q", i, firstDownloader.RpcType, expected.FirstDownloaderType)
						}
						// Check the constructed RpcUrl
						if expected.FirstDownloaderRpcUrl != "" && firstDownloader.RpcUrl != expected.FirstDownloaderRpcUrl {
							t.Errorf("Task %d, Downloader 0: got RpcUrl %q, want %q", i, firstDownloader.RpcUrl, expected.FirstDownloaderRpcUrl)
						}
						// Remove checks for deprecated fields Url and Host
					}
					expectedInterval := time.Duration(expected.FetchIntervalMinutes) * time.Minute
					if task.FetchInterval != expectedInterval {
						t.Errorf("Task %d: got interval %v, want %v", i, task.FetchInterval, expectedInterval)
					}
				}
			}

		})
	}
}

func TestFeedConfig(t *testing.T) {
	tests := []struct {
		name    string
		yamlStr string
		want    []string
	}{
		{
			name:    "single string",
			yamlStr: `feed: "http://example.com/feed1"`,
			want:    []string{"http://example.com/feed1"},
		},
		{
			name: "multi-line array",
			yamlStr: `
feed:
  - http://example.com/feed1
  - http://example.com/feed2`,
			want: []string{"http://example.com/feed1", "http://example.com/feed2"},
		},
		{
			name:    "inline array",
			yamlStr: `feed: ["http://example.com/feed1", "http://example.com/feed2"]`,
			want:    []string{"http://example.com/feed1", "http://example.com/feed2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg struct {
				Feed FeedConfig `yaml:"feed"` // Field name must match YAML key
			}
			if err := yaml.Unmarshal([]byte(tt.yamlStr), &cfg); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if len(cfg.Feed.URLs) != len(tt.want) {
				t.Fatalf("Got %d URLs, want %d", len(cfg.Feed.URLs), len(tt.want))
			}
			for i := range tt.want {
				if cfg.Feed.URLs[i] != tt.want[i] {
					t.Errorf("URL[%d] = %q, want %q", i, cfg.Feed.URLs[i], tt.want[i])
				}
			}
		})
	}
}
