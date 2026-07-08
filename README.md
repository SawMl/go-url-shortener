# shorty

> **Status:** Phase 2 — Postgres persistence

A URL shortener service written in **Go**. REST API with persistent storage and expandable architecture.

## What it does

```
POST /shorten   {"url":"https://a-very-long-example.com/path"}
        →  {"code":"1","short":"http://localhost:8080/1"}

GET  /1   →  302 redirect  →  https://a-very-long-example.com/path
```

## Run locally

Requires Go 1.22+.

### With in-memory storage (default)

```sh
go run .
```

### With Postgres

```sh
# Install Postgres (or use Docker)
docker run -d -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:latest

# Set the connection string and run
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
go run .
```

The database table is created automatically on startup.

Then in another terminal:

```sh
# shorten a URL
curl -X POST localhost:8080/shorten -d '{"url":"https://example.com"}'

# follow the redirect
curl -iL localhost:8080/1
```

## Run tests

```sh
go test ./...
```

## Architecture (current)

```
                           ┌─────────────────┐
client ──POST /shorten──▶ │ HTTP Handler    │
client ──GET /{code}────▶ │                 │
                           └────────┬────────┘
                                    │
                              Store │ (interface)
                                    │
                    ┌───────────────┴────────────────┐
                    │                                │
        ┌───────────▼──────────┐        ┌──────────▼──────────┐
        │  memoryStore         │        │  postgresStore      │
        │  (in-memory map)     │        │  (PostgreSQL)       │
        │  ▸ Save()            │        │  ▸ Save()           │
        │  ▸ Lookup()          │        │  ▸ Lookup()         │
        └──────────────────────┘        └─────────────────────┘
```

The `Store` interface allows swapping backends without changing handlers.
- Default: in-memory (lose data on restart)
- With `DATABASE_URL`: Postgres (persistent)

## Roadmap

- [x] **Phase 1** — In-memory shorten + redirect, base62 codes, unit tests
- [x] **Phase 2** — Postgres persistence, Store interface, migrations, input validation
- [ ] **Phase 3** — Redis caching, per-IP rate limiting, hit metrics
- [ ] **Phase 4** — GitHub Actions CI, Dockerfile, load-test benchmark
- [ ] **Phase 5** — Deploy to public URL, architecture writeup

## Tech stack

Go · Postgres · Redis · Docker · GitHub Actions
