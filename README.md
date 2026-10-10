# GoRSS

A self-hosted RSS/Atom reader written in Go with an embedded web UI. Built for zero external runtime dependencies, and simple deployment on a home server via Docker.


## Features

- **Self-Contained Single Binary**: HTML templates, CSS, and assets are embedded via Go's standard `embed.FS`. No Node.js build step or external web server required.
- **Fast, Reactive UI**: Server-side rendered with HTMX for partial page swaps (marking articles read/unread, toggling stars, and managing subscriptions without full page reloads).
- **SQLite Storage**: Uses WAL (Write-Ahead Logging) mode, busy timeouts, and cascading deletes (`ON DELETE CASCADE`) to keep feed cleanup atomic.
- **Smart Background Polling**: Periodic worker loop fetches updates using HTTP conditional caching (`ETag` and `If-Modified-Since`) to avoid downloading unchanged feeds (HTTP 304).
- **Format Normalization**: Standardizes RSS 0.9x/2.0 and Atom 1.0 feeds into a unified schema.
- **OPML Support**: Export and import your feed list for compatibility with NetNewsWire, Reeder, Feedly, and Miniflux.
- **JSON REST API**: Full programmatic access to feeds and articles.

## Dependencies

- Go 1.22+ (if running natively)
- SQLite3 build headers (CGO enabled for mattn/go-sqlite3)
- Docker & Docker Compose (for containerized deployment)

## Run locally

Clone the repo
```bash
git clone [https://github.com/Tomassalgueiro/gorss.git](https://github.com/Tomassalgueiro/gorss.git)
cd gorss
```

Download dependencies
```bash
go mod download
```

Run the program
```bash
go run cmd/server/main.go 
```

Open the browser at http://localhost:6060


## Deploy it on docker

## Deployment with Docker

The container runs as an unprivileged user and stores the SQLite database in a mounted persistent volume.

### Using Docker Compose

**Start the service**:
```bash
docker compose up -d
```
**Check the logs at**:
```bash
docker compose logs -f gorss 
```

```bash
docker compose down
```

The data is storade locally under the "./data" (./data/gorss.db)

## Configuration

Settings can be customized via environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `6060` | HTTP port the server listens on |
| `DATABASE_URL` | `gorss.db` | File path to the SQLite database |
| `POLL_INTERVAL` | `5m` | Frequency of background feed updates |
| `WORKER_POOL_SIZE` | `5` | Maximum concurrent HTTP feed fetches |
