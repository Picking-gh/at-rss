# at-rss: RSS Feed Monitor for Aria2c & Transmission

A lightweight RSS feed monitor that watches RSS feeds and automatically
sends torrent/magnet links to Aria2c or Transmission via JSON-RPC.

## Features

- Aria2c and Transmission RPC with automatic session-id renewal
- Multiple downloaders per task with failover (tries in order)
- Flexible keyword filtering: include/exclude with AND/OR logic
- Magnet link extraction from any RSS tag via regex (trick mode)
- Character-level traditional → simplified Chinese conversion for anime feeds
- Persistent cache to avoid duplicate downloads
- Automatic cleanup of completed/stopped downloads
- Hot-reload: edit the YAML config, the daemon picks it up without restart
- Web UI with REST API for managing tasks
- Bearer-token authentication for API

## Installation

Requires Go 1.23+.

```bash
git clone https://github.com/picking-gh/at-rss.git
cd at-rss

# Build the daemon
go build

# Build the Web UI
cd webui
npm install && npm run build
cd ..
```

## Configuration

Create `at-rss.conf` in YAML format. For a complete reference, see
`at-rss.conf.example`.

```yaml
# Each top-level key is a task name (display only)
my_anime:
  downloaders:
  - type: aria2c
    token: "my-secret-token"
    autoCleanUp: true
  - type: transmission           # fallback if aria2c fails
    host: "nas.local"

  feeds:
  - "https://example.com/rss/1080p"
  - "https://example.com/rss/720p"

  filter:
    include:
    - "1080p, hevc"              # AND: must contain both
    - "720p, x264"               # OR: this group too
    exclude:
    - "batch"

  extracter:                     # optional: extract infohash from a tag
    tag: link
    pattern: "[0-9a-f]{40}"

  interval: 30                   # minutes (default: 10)
```

### Downloader reference

| Field       | Aria2c default | Transmission default    |
|-------------|---------------|-------------------------|
| `host`      | `localhost`   | `localhost`             |
| `port`      | `6800`        | `9091`                  |
| `rpcPath`   | `/jsonrpc`    | `/transmission/rpc`     |
| `useHttps`  | `false`       | `false`                 |
| Auth        | `token`       | `username` + `password` |

### Filter reference

- `include`: list of keyword groups. Items match if **any** group matches.
- `exclude`: list of keyword groups. Items are skipped if **any** group matches (exclude wins).
- Keywords within a group are comma-separated and have **AND** semantics.
- Matching is case-insensitive and traditional→simplified normalized.
- Omitting the `filter` section entirely means download everything.
- An empty `include` list means download everything (explicit intent).
- `include: [""]` (single empty string) means download nothing — used as a
  placeholder by the Web UI when downloads are paused.

### Extractor reference

When `extracter` is present, the daemon extracts a hash from the specified
RSS `tag` using `pattern` (regex with one capture group), then constructs
a `magnet:?xt=urn:btih:...` URI. Valid tags: `title`, `link`, `description`,
`enclosure`, `guid`.

## Running

```bash
# Basic usage
./at-rss -c /path/to/at-rss.conf

# With Web UI
./at-rss -c /path/to/at-rss.conf --web-listen :8080 --web-ui-dir webui/dist

# With API auth
./at-rss -c /path/to/at-rss.conf --web-listen :8080 --token my-secret
```

Or run as a systemd service (see `at-rss.service`).

## CLI

```
Usage: at-rss [OPTIONS]

  -c, --conf <PATH>              Config file (default: at-rss.conf)
      --web-listen <ADDR>        Listen address (e.g. ':8080')
      --web-ui-dir <DIR>         Web UI static files (default: webui/dist)
      --token <TOKEN>            API auth token (empty = no auth)
      --default-fetch-interval <N>  Default interval in minutes (default: 0)
```

## REST API

All `/api/*` routes require `Authorization: Bearer <TOKEN>` if `--token` is set.

| Method   | Path                 | Description    |
|----------|----------------------|----------------|
| `GET`    | `/api/tasks`         | List all tasks |
| `POST`   | `/api/tasks`         | Create a task  |
| `GET`    | `/api/tasks/{name}`  | Get one task   |
| `PUT`    | `/api/tasks/{name}`  | Update a task  |
| `DELETE` | `/api/tasks/{name}`  | Delete a task  |

Credentials in downloader configs are masked (`******`) in API responses.
Config changes take effect on the next polling cycle without restart.

## Cache

Processed feed items are tracked in `~/.cache/at-rss.json`. The cache is flushed
after each fetch cycle. Entries older than 30 days with no info-hash data are
automatically cleaned up.

## License

MIT — see [LICENSE](LICENSE).
