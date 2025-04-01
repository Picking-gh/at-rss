package main

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigParsing(t *testing.T) {
	tests := []struct {
		name    string
		yamlStr string
		wantErr bool
		errMsg  string
	}{
		{
			name: "single feed with aria2c",
			yamlStr: `feed1:
  aria2c:
    url: "ws://localhost:6800/jsonrpc"
    token: "abcd"
  feed: "http://example.com/feed1"
  interval: 30`,
			wantErr: false,
		},
		{
			name: "multi-line feed with transmission",
			yamlStr: `feed2:
  transmission:
    host: "localhost"
    port: 9091
  feed:
    - http://example.com/feed1
    - http://example.com/feed2`,
			wantErr: false,
		},
		{
			name: "with filter and extracter",
			yamlStr: `feed3:
  aria2c:
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
			name: "default values",
			yamlStr: `feed4:
  aria2c: {}
  feed: "http://example.com/feed4"`,
			wantErr: false,
		},
		{
			name: "invalid config - both rpc servers",
			yamlStr: `feed5:
  aria2c: {}
  transmission: {}
  feed: "http://example.com/feed5"`,
			wantErr: true,
			errMsg:  "cannot specify both aria2c and transmission",
		},
		{
			name: "invalid config - no feed",
			yamlStr: `feed6:
  aria2c: {}`,
			wantErr: true,
			errMsg:  "must specify at least one feed URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := yaml.Unmarshal([]byte(tt.yamlStr), &cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain %q, got %q", tt.errMsg, err.Error())
			}
			if !tt.wantErr && len(cfg.Tasks) == 0 {
				t.Error("Expected tasks but got none")
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
			yamlStr: `feed:
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
				Feed FeedConfig `yaml:"feed"`
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
