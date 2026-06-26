# Enrichment Service

Go HTTP service that simulates profile enrichment, stores enriched profiles in Postgres, and exposes endpoints for starting enrichment jobs and reading saved profile data.

## Requirements

- Go 1.26
- Docker and Docker Compose
- Optional: `migrate` CLI if you want to run migrations manually
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

The `-d` flag means detached mode, so containers run in the background.

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
postman/Profile Enrichment Service.postman_collection.json
```

Import it into Postman, start the service, then run:

- `enrich`: `POST http://localhost:8080/v1/enrich`
- `profiles`: `GET http://localhost:8080/v1/profiles/p1`

Run `enrich` first so there is profile data available for `profiles`.

## API

### Enrich Profiles

```powershell
curl -X POST http://localhost:8080/v1/enrich `
  -H "Content-Type: application/json" `
  -d '{"profile_ids":["p1","p2","p3"]}'
```

Example response:

```json
{
  "requested": 3,
  "enriched": 3,
  "failed": 0
}
```

If upstream simulation fails for some profiles, the response includes failures:

```json
{
  "requested": 3,
  "enriched": 2,
  "failed": 1,
  "failures": [
    {
      "profile_id": "p2",
      "reason": "upstream: simulated failure for profile \"p2\""
    }
  ]
}
```

### Get Profile

```powershell
curl http://localhost:8080/v1/profiles/p1
```

Example response:

```json
{
  "id": "p1",
  "username": "User p1",
  "email": "p1@mail.com",
  "enriched_at": "2026-06-26T12:00:00Z"
}
```

## Tests

Run all tests:

```powershell
go test ./...
```
