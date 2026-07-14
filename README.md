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

**Option 1: Using docker-compose (recommended)**

```sh
docker-compose up -d
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/shorty?sslmode=disable"
go run .
```

**Option 2: Manual Docker**

```sh
docker run -d -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:latest
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
go run .
```

The database table is created automatically on startup.

### Fully containerized (shorty + Postgres)

Build and run the entire stack — no local Go toolchain required:

```sh
docker-compose up --build
```

This builds the `shorty` image from the [Dockerfile](Dockerfile), starts Postgres,
waits for it to be healthy, then launches shorty wired to the database.
The API is available at `http://localhost:8080`.

To build the image on its own:

```sh
docker build -t shorty .
```

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

## Input Validation

The `/shorten` endpoint validates URLs before storing:

- **Scheme:** Must be `http://` or `https://` only
- **Structure:** Valid URL format (parsed by `net/url.Parse`)
- **Host:** Must have a hostname, not just a scheme
- **Size:** Max 2048 characters (DOS prevention)
- **Localhost prevention:** Rejects `localhost`, `127.0.0.1`, `::1` (redirect loop prevention)
- **Private IPs:** Rejects `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16` (internal-only IPs)

Invalid URLs return `400 Bad Request`.

## CI/CD

GitHub Actions automatically runs on every push:
- **Tests** — `go test ./...`
- **Vet** — `go vet ./...` (code quality checks)
- **Fmt** — `gofmt` (code formatting)

View results in the [Actions](https://github.com/SawMl/go-url-shortener/actions) tab.

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
- [x] **Phase 3** — GitHub Actions CI (auto-test, vet, fmt on push)
- [x] **Phase 4** — Input validation hardening, comprehensive tests
- [x] **Phase 5** — Dockerfile (containerize shorty itself), full docker-compose stack
- [ ] **Phase 6** — Redis caching, per-IP rate limiting, hit metrics
- [ ] **Phase 7** — Load-test benchmark, deployment

## Tech stack

Go · Postgres · Redis · Docker · GitHub Actions
