# Project File Structure & Responsibilities

This document defines the baseline directory layout for the RSS reader project and details the role of each file and folder.

.
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   ├── db.go
│   │   └── schema.sql
│   ├── feed/
│   │   ├── client.go
│   │   ├── model.go
│   │   ├── parser.go
│   │   └── repository.go
│   ├── server/
│   │   ├── handlers.go
│   │   ├── routes.go
│   │   └── server.go
│   └── worker/
│       ├── pool.go
│       └── scheduler.go
├── web/
│   ├── static/
│   │   └── style.css
│   └── templates/
│       ├── base.html
│       └── index.html
├── .dockerignore
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md

Core Entry Point

    cmd/server/main.go

    Application root entrypoint. Parses flags or environment variables, initializes foundational resources (logger, database, config), instantiates components, wires dependencies, starts background workers and the HTTP server, and traps OS signals (SIGINT, SIGTERM) for graceful teardown.

Internal Packages (internal/)

Code inside internal/ cannot be imported by external Go projects, enforcing strict domain isolation.
Configuration (internal/config/)

    internal/config/config.go

    Loads and validates runtime settings (server port, SQLite file path, refresh interval, logging level) from environment variables or flags, parsing them into a typed Go configuration struct.

Storage & Schema (internal/database/)

    internal/database/db.go

    Manages the raw *sql.DB connection pool lifecycle. Configures SQLite-critical pragmas (enabling WAL mode, busy timeouts, and foreign key enforcement).

    internal/database/schema.sql

    Contains raw SQL DDL statements creating tables, indices, and relationships (feeds, articles, tabs, tab_feeds).

Feed Processing & Persistence (internal/feed/)

    internal/feed/model.go

    Declares the core domain structures: Feed, Article, and Tab, along with custom types or enums (such as read status or subscription state).

    internal/feed/repository.go

    Implements the persistence layer for feeds, articles, and tabs. Encapsulates all raw SQL queries (INSERT, UPDATE, SELECT, JOIN) away from the HTTP and worker layers.

    internal/feed/client.go

    An HTTP wrapper responsible for fetching feeds. Configures strict outbound network timeouts, custom User-Agent headers, redirect limits, and conditional caching headers (ETag, If-Modified-Since).

    internal/feed/parser.go

    Accepts raw XML byte streams and normalizes RSS 0.9x/2.0, Atom 1.0, and RDF formats into standard domain models. Handles GUID resolution, timestamp parsing, and malformed XML.

HTTP Routing & Handlers (internal/server/)

    internal/server/server.go

    Defines the HTTP server struct, registers global middleware (logging, panic recovery, security headers), and implements start/stop lifecycle methods.

    internal/server/routes.go

    Maps URL patterns (e.g., GET /, POST /feeds, PUT /articles/{id}/read) to their corresponding handler functions using standard library routing.

    internal/server/handlers.go

    Contains request handlers. Parses form inputs and URL parameters, calls the domain repository, and renders HTML templates (or JSON responses) back to the client.

Concurrency & Scheduler (internal/worker/)

    internal/worker/pool.go

    Defines the fixed-size worker pool. Distributes fetch jobs across a controlled number of goroutines using channels, preventing socket exhaustion.

    internal/worker/scheduler.go

    Periodically scans the database for feeds that need updating, generates jobs, pushes them into the worker pool, and handles scheduled teardowns via context.Context.

Presentation Layer (web/)

    web/templates/base.html

    The main HTML layout skeleton, containing the , , global styles, and script tags (including HTMX).

    web/templates/index.html

    The main dashboard template displaying the custom tab bar, feed listings, and article list.

    web/static/style.css

    Minimalist stylesheet for the user interface.

Project Metadata & Infrastructure

    go.mod & go.sum

    Go module definitions and cryptographically locked dependency checksums.

    .gitignore

    Excludes compiled binaries (bin/), test coverage artifacts, SQLite files (*.db), and temporary files from version control.

    Dockerfile & docker-compose.yml

    Multi-stage build definition to produce a minimal, non-root scratch/Alpine container image and run it alongside local storage volumes.

    README.md

    Project overview, setup instructions, architecture notes, and configuration documentation.
