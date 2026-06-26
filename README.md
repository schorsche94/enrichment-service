# Enrichment Service

Go HTTP service that simulates profile enrichment, stores enriched profiles in Postgres, and exposes endpoints for starting enrichment jobs and reading saved profile data.

## Requirements

- Go 1.26
- Docker and Docker Compose
- Optional: Postman for the included API collection

## Configuration

The service reads configuration from environment variables.

| Variable | Default | Description |
| --- | --- | --- |
| `ADDR` | `:8080` | HTTP listen address |
| `CONCURRENCY` | `5` | Maximum concurrent profile enrichment jobs |
| `DB_ADDR` | `postgres://enricher_user:enricher_pwd@localhost:5436/enrich?sslmode=disable` | Postgres connection string |
| `DB_MAX_OPEN_CONNS` | `30` | Maximum open DB connections |
| `DB_MAX_IDLE_CONNS` | `30` | Maximum idle DB connections |
| `DB_MAX_IDLE_TIME` | `15m` | Maximum idle lifetime for DB connections |

## Run With Docker Compose

```powershell
docker compose up -d --build
```

This builds the API image, starts Postgres, applies migrations, and runs the API at `http://localhost:8080`.

View logs:

```powershell
docker compose logs -f
```

Stop everything:

```powershell
docker compose down
```

## Run Locally

Start Postgres and migrations:

```powershell
docker compose up -d db migrate-reset
```

Run the API:

```powershell
go run ./cmd/api
```

## Postman

A Postman collection is included at:

```text
requests/Profile Enrichment Service.postman_collection.json
```

Import it into Postman, start the service, then run:

- `enrich`: `POST http://localhost:8080/v1/enrich`
- `profiles`: `GET http://localhost:8080/v1/profiles/p1`

Run `enrich` first so there is profile data available for `profiles`.

If you're using the licensed version of GoLand, you can run the HTTP requests from requests/requests.http.

## Tests

Run all tests:

```powershell
go test ./...
```

## Future Work

- Retry failed upstream calls with exponential backoff and a cap.
- An in-memory cache so repeat IDs within a short window skip the upstream.
- More integration coverage around database migrations, concurrent enrichment requests, and partial upstream failures.
- Observability hooks such as structured request logs, latency metrics, and counters for upstream failures.
