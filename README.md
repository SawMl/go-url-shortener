# shorty

> **Status:** 🚧 In active development (Phase 1 — walking skeleton)

A URL shortener service written in **Go**. REST API with caching and rate limiting (roadmap below).

## What it does

```
POST /shorten   {"url":"https://a-very-long-example.com/path"}
        →  {"code":"1","short":"http://localhost:8080/1"}

GET  /1   →  302 redirect  →  https://a-very-long-example.com/path
```

## Run locally

Requires Go 1.22+.

```sh
go run .
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

## Architecture (current)

```
client ──POST /shorten──▶ handler ──▶ in-memory store (map + counter)
client ──GET /{code}───▶ handler ──▶ 302 redirect
```

Phase 1 keeps everything in memory so the redirect loop works end-to-end first.
Persistence, caching, and rate limiting are layered in next.

## Roadmap

- [x] **Phase 1** — In-memory shorten + redirect, base62 codes, unit tests
- [ ] **Phase 2** — Postgres persistence, input validation, GitHub Actions CI, Dockerfile
- [ ] **Phase 3** — Redis caching, per-IP rate limiting, hit metrics, load-test benchmark
- [ ] **Phase 4** — Deploy to a public URL, architecture diagram, writeup

## Tech stack

Go · Postgres · Redis · Docker · GitHub Actions
