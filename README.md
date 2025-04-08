# at-rss: A RSS Feed Parser for aria2c and transmission

A lightweight RSS feed parser that monitors RSS feeds and automatically downloads torrents via aria2c or transmission RPC.

## Features

- Supports both aria2c and transmission RPC
- Flexible feed configuration with include/exclude filters
- Magnet link extraction and reconstruction
- Automatic cleanup of completed downloads
- Simple YAML configuration

## Installation

1. Install Go 1.20+ 
2. Clone this repository:
   ```bash
   git clone https://github.com/picking-gh/at-rss.git
   cd at-rss
   ```
3. Build the binary:
   ```bash
   go build
   ```

## Configuration

Create `at-rss.conf` in YAML format:

```yaml
# Example configuration
my_feed:
  downloader:
    - type: aria2c
      token: "your_token"
      autoCleanUp: true
  feed: "https://example.com/rss"
  filter:
    include: ["1080p", "x264"]
    exclude: ["camrip", "tc"]
  interval: 60 # minutes
```

## Running

```bash
./at-rss -c /path/to/at-rss.conf
```

Or run as a systemd service (see at-rss.service for example).

## Configuration Options

| Key          | Required | Description                          |
|--------------|----------|--------------------------------------|
| downloader   | Yes      | Download client configuration        |
| feed         | Yes      | List of RSS feed URLs                |
| filter.include | No     | Keywords to include                  |
| filter.exclude | No     | Keywords to exclude                  |
| interval     | No       | Polling interval in minutes (default: 10) |
| extracter    | No       | Magnet link extraction configuration |

## Technical Details

- Uses gofeed for RSS parsing
- Supports Chinese text conversion (simplified/traditional)
- Implements caching to avoid duplicate downloads

## License

MIT License - See [LICENSE](LICENSE) for details.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss.
