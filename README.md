# Orders API

A small, production-quality Orders API in Go, running entirely in Docker Compose.
Stack: Go + [chi](https://github.com/go-chi/chi) router, [pgx](https://github.com/jackc/pgx) (pgxpool)
for Postgres, [go-redis](https://github.com/redis/go-redis) for Redis, `log/slog` JSON logging,
plain SQL migrations.

This is the application layer of the observability lab. Prometheus, Grafana, tracing,
load tests, and incident experiments are intentionally **not** included yet.
See [docs/lab-overview.md](docs/lab-overview.md) for the full lab vision.

## Requirements

- Docker with Compose v2
- `make` (optional, commands are listed below)
- Go 1.25+ (only for running tests locally)

## Quick start

```bash
cp .env.example .env      # then edit POSTGRES_PASSWORD
make up                   # build image, start postgres + redis + api
make seed                 # 10,000 customers, 1,000,000 orders (~25s)
curl localhost:8080/health
```

Other targets:

```bash
make logs     # follow api logs
make test     # go test ./...
make down     # stop containers
make clean    # stop containers and delete the Postgres volume
```

Migrations in `migrations/` run automatically when the API starts (tracked in
`schema_migrations`, guarded by an advisory lock). The seed command also runs them.

## Configuration

Set in `.env` (see [.env.example](.env.example)). Never commit `.env`.

| Variable | Default | Description |
|---|---|---|
| `HTTP_PORT` | `8080` | Host port the API is published on |
| `DATABASE_URL` | — | Postgres DSN (required by seed run on the host) |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `DB_MAX_CONNS` | `10` | pgxpool max connections |
| `CACHE_TTL_SECONDS` | `300` | TTL for cached orders |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | `orders` / — / `orders` | Postgres bootstrap credentials |

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/orders` | Validate, insert into Postgres, invalidate cache, `201` |
| `GET` | `/orders/{id}` | Cache-aside: Redis first, Postgres on miss, `404` if absent |
| `GET` | `/customers/{id}/orders` | Last 50 orders, `created_at DESC` |
| `GET` | `/health` | Liveness, always `200` |
| `GET` | `/ready` | Pings Postgres + Redis, `200` or `503` |

Request validation on `POST /orders`: `customer_id > 0`, `total_cents > 0`,
`status` ∈ `pending | paid | shipped | cancelled | refunded`, and the customer must
exist (FK), otherwise `400`.

## Curl examples

```bash
# Create an order
curl -X POST localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer_id":1,"total_cents":5000,"status":"pending"}'
# 201 {"id":1000001,"customer_id":1,"status":"pending","total_cents":5000,"created_at":"..."}

# Read it back (2nd call served from Redis)
curl localhost:8080/orders/1000001

# Validation failures
curl -X POST localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer_id":1,"total_cents":5000,"status":"nope"}'
# 400 {"error":"invalid order status"}

# Order history
curl localhost:8080/customers/7/orders

# Health / readiness
curl localhost:8080/health
curl localhost:8080/ready
```

## Seeding

```bash
make seed                 # refuses if data exists
make seed ARGS=--force    # truncate and reload
```

Generates 10,000 customers and 1,000,000 orders with two bulk
`INSERT ... SELECT generate_series` statements; `created_at` values are spread over
the last 730 days.

## Tests

```bash
make test     # go test ./...
```

Handler-level unit tests with mocked store/cache interfaces (no database needed).

## Layout

```
cmd/api          HTTP server entrypoint
cmd/seed         bulk data loader
internal/config  env config with defaults
internal/handler HTTP handlers + tests
internal/middleware  JSON request logger (chi route patterns)
internal/model   domain types
internal/server  router wiring
internal/store   OrderStore/Cache interfaces, pgx + go-redis impls
migrations       plain SQL migrations, embedded, auto-applied
```
