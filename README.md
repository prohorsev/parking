# Smart Parking Aggregator API

A Go backend service that aggregates real-time parking data from multiple providers,
normalises it into a unified schema, stores it in PostgreSQL, and exposes a clean REST API.

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                          HTTP Clients                            │
└──────────────────────────────┬───────────────────────────────────┘
                               │  GET /parking?lat=&lng=
                               │  GET /parking/{id}
                         ┌─────▼──────┐
                         │  chi Router │
                         └─────┬──────┘
                               │
                         ┌─────▼──────────┐
                         │   Aggregator   │  ← cache-first read
                         └──┬────────┬───┘
                            │        │
              ┌─────────────▼─┐  ┌───▼────────────┐
              │  In-Memory    │  │   Repository   │
              │  Cache (TTL)  │  │   (Postgres)   │
              └───────────────┘  └───────┬────────┘
                                         │ upsert
                               ┌─────────▼──────────┐
                               │   Background Loop  │  every 60 s
                               └──────┬─────────────┘
                                      │  FetchAll()
                          ┌───────────┴───────────┐
                   ┌──────▼──────┐       ┌─────────▼──────┐
                   │ Provider A  │       │  Provider B    │
                   │ GET /lots   │       │ GET /parking-  │
                   │ (mock HTTP) │       │ zones (mock)   │
                   └─────────────┘       └────────────────┘
```

### Key design decisions

| Concern | Approach |
|---|---|
| Multi-provider integration | `provider.Provider` interface — swap/add sources without touching the aggregator |
| Schema normalisation | Each provider package owns its raw structs and maps them to `domain.ParkingLot` |
| Persistence | PostgreSQL with `UNIQUE(external_id, provider)` upsert — idempotent refreshes |
| Caching | In-memory TTL cache in front of Postgres — zero external deps for caching |
| Geo search | Bounding-box query + Haversine sort — no PostGIS required |
| Background refresh | Goroutine loop; first run is synchronous before the HTTP server starts accepting |
| Graceful shutdown | `os.Signal` + `http.Server.Shutdown` with 5 s deadline |

---

## Project layout

```
.
├── cmd/api/            # main entry point
├── internal/
│   ├── api/            # HTTP router + handlers (chi)
│   ├── aggregator/     # orchestration: fetch → normalise → cache → serve
│   ├── cache/          # thread-safe in-memory TTL cache
│   ├── config/         # env-based configuration
│   ├── domain/         # shared domain model (ParkingLot, SearchParams)
│   ├── provider/
│   │   ├── provider.go     # Provider interface
│   │   ├── providera/      # Mock server + normalising client for Provider A
│   │   └── providerb/      # Mock server + normalising client for Provider B
│   └── repository/
│       ├── repository.go   # Repository interface
│       └── postgres/       # PostgreSQL implementation
├── migrations/         # reference SQL (schema applied automatically on startup)
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

---

## Quick start

### Option A — Docker Compose (recommended)

```bash
docker compose up --build
```

The API will be available at `http://localhost:8080` once the Postgres health check passes.

### Option B — Local (requires Postgres)

```bash
# 1. Start Postgres
docker compose up -d postgres

# 2. Run the API
make run
```

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | TCP address for the HTTP server |
| `DATABASE_URL` | `postgres://parking:parking@localhost:5432/parking?sslmode=disable` | Postgres DSN |
| `PROVIDER_A_PORT` | `9001` | Port for the in-process Provider A mock server |
| `PROVIDER_B_PORT` | `9002` | Port for the in-process Provider B mock server |
| `CACHE_TTL` | `30s` | In-memory cache TTL for query results |
| `REFRESH_INTERVAL` | `60s` | How often the background loop re-fetches from providers |

---

## API reference

### `GET /health`

Returns `200 OK` when the service is running.

```json
{"status": "ok"}
```

---

### `GET /parking`

Search for parking lots near a coordinate.

**Query parameters**

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `lat` | float | ✓ | — | Latitude of search origin |
| `lng` | float | ✓ | — | Longitude of search origin |
| `radius` | float | | `5.0` | Search radius in kilometres |

**Example**

```bash
curl "http://localhost:8080/parking?lat=55.7558&lng=37.6173&radius=5"
```

**Response `200 OK`**

```json
{
  "data": [
    {
      "id": "018e4b7c-...",
      "external_id": "a-001",
      "provider": "provider_a",
      "name": "City Centre Garage",
      "address": "1 Red Square",
      "lat": 55.7558,
      "lng": 37.6173,
      "total_spots": 150,
      "free_spots": 42,
      "price_per_hour": 2.50,
      "currency": "USD",
      "updated_at": "2026-03-22T10:00:00Z",
      "distance_km": 0.0
    }
  ],
  "count": 1,
  "params": {
    "lat": 55.7558,
    "lng": 37.6173,
    "radius_km": 5.0
  }
}
```

---

### `GET /parking/{id}`

Retrieve a single parking lot by its UUID.

**Example**

```bash
curl "http://localhost:8080/parking/018e4b7c-abcd-1234-..."
```

**Response `200 OK`**

```json
{
  "data": {}
}
```

**Response `404 Not Found`**

```json
{"error": "parking lot not found"}
```

---

## Provider schema comparison

The aggregator normalises two deliberately different schemas into a single `ParkingLot` model:

| Field | Provider A | Provider B | Domain |
|---|---|---|---|
| ID | `parking_id` | `uid` | `external_id` |
| Name | `name` | `title` | `name` |
| Coordinates | `location.lat` / `location.lon` | `geo` ("lat,lng" string) | `lat` / `lng` |
| Capacity | `total_capacity` | `spaces_total` | `total_spots` |
| Availability | `available_spots` | `spaces_free` | `free_spots` |
| Price | `hourly_rate` | `price` | `price_per_hour` |
| Currency | `currency_code` | `price_currency` | `currency` |
| Address | `street` | `full_address` | `address` |
