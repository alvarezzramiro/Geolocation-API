# Geolocation API
 
A production-grade routing API built in Go that calculates optimal routes between street intersections using real OpenStreetMap data. The project demonstrates graph database modeling, two pathfinding algorithms with live performance comparison, and a layered architecture with caching.
 
Built as a personal project to explore technologies outside the traditional Java/Python stack.
 
---
 
## Tech Stack
 
| Layer | Technology | Why |
|---|---|---|
| Language | Go | Systems-level performance, native concurrency, minimal runtime |
| Graph database | Neo4j | Streets and intersections are a native graph DB |
| Cache | Redis | Sub-millisecond route retrieval for repeated queries |
| Map data | OSM via Overpass API | Real street data with names, directions and speed limits. Downloaded automatically at startup |
Router | chi | Lightweight HTTP router, idiomatic Go |
Containerization | Docker + Docker Compose | One-command setup for the full stack

---

## Architecture

```
Geolocation-API/
├── api/                  # HTTP handlers, router, request parsing
├── config/               # Environment-based configuration
├── db/                   # Database connections (Neo4j, Redis)
├── graph/                # Pathfinding algorithms and graph loading
│   ├── types.go          # Shared types: Graph, Edge, Result, NodeCoords
│   ├── router.go         # Router interface + algorithm selector
│   ├── loader.go         # LoadGraph — reads Neo4j into memory at startup
│   ├── dijkstra.go       # Dijkstra implementation
│   ├── astar.go          # A* implementation with Haversine heuristic
│   └── path.go           # Path reconstruction + CompressSteps
├── internal/
│   └── text/             # Shared utilities (accent normalization)
├── repository/           # Business queries (route cache, intersection lookup)
├── seed/                 # OSM data fetching and Neo4j loading
├── Dockerfile
├── docker-compose.yml
└── main.go
```

---

### How it works

At startup the app downloads the street map of central Tandil from OpenStreetMap, loads it into Neo4j as a directed weighted graph, and copies the full graph into memory. Neo4j is only consulted again for intersection name lookups — all routing runs in-memory for maximum performance.
 
Each intersection is a node. Each street segment is a directed ROAD relationship weighted by travel time in seconds (distance / speed). Two-way streets generate two relationships in opposite directions.
 
Routes are cached in Redis per algorithm, so repeated queries return in microseconds without re-running the pathfinding.

---

## Endpoints

### `GET /streets`
Returns all street names in the loaded map

Query params: `q` (optional) — partial name filter, case and accent insensitive.

```
GET /streets?q=av
```
```json
["Avenida Perón", "Avenida Rivadavia", "Avenida Santamarina"]
```

---

### `GET /nodes`
Returns all intersection nodes with coordinates and names.

```
GET /nodes
```
```json
[
  {
    "id": "osm_448193029",
    "name": "Chacabuco & Avenida España",
    "type": "intersection",
    "lat": -37.3237,
    "lon": -59.1414
  }
]
```

### `GET /route/by-intersection`
Calculates the optimal route between two named intersections.

**Query params:**
- from — origin intersection in `Street A/Street B` format
- to — destination intersection in `Street A/Street B` format
- algorithm — dijkstra (default) or astar

```
GET /route/by-intersection?from=chacabuco/25 de mayo&to=sarmiento/14 de julio
```
```json
{
  "algorithm": "dijkstra",
  "resolved": {
    "from": "Chacabuco & 25 de Mayo",
    "to": "Sarmiento & 14 de Julio"
  },
  "result": {
    "NodesVisited": 349,
    "Steps": [
      {
        "from": "Chacabuco & 25 de Mayo (osm_448193404)",
        "street": "25 de Mayo",
        "to": "14 de Julio & 25 de Mayo (osm_448193402)"
      },
      {
        "from": "14 de Julio & 25 de Mayo (osm_448193402)",
        "street": "14 de Julio",
        "to": "Sarmiento & 14 de Julio (osm_448193264)"
      }
    ],
    "TotalMins": 2.7,
    "TotalSecs": 159.8
  },
  "source": "computed"
}
```

The NodesVisited field shows how many graph nodes each algorithm explored. A* consistently visits fewer nodes than Dijkstra for the same route, demonstrating the effect of the geographic heuristic.

---

### `GET /route`
Calculates a route by node ID (for programmatic use).

**Query params:** `from`, `to` (OSM node IDs), `algorithm`

```
GET /route?from=osm_448193431&to=osm_448193211&algorithm=dijkstra
```

---

## Dijkstra vs A*

Both algorithms are implemented and selectable at request time. They always return the same optimal route. The difference is efficiency.

Dijkstra explores all reachable nodes in order of accumulated cost, with no sense of direction toward the destination. On a 3,242-node graph it may visit over 1,800 nodes for a single route.

A* adds a geographic heuristic — the straight-line Haversine distance to the destination converted to seconds at maximum urban speed. This guides the search toward the destination and typically reduces nodes visited by 80–90% for the same result.

The heuristic is admissible: it never overestimates the remaining cost because no real route can be shorter than a straight line or faster than the maximum speed.

** COMPARISION **

```
route/by-intersection?from= chacabuco / las heras &to= avellaneda / falucho
```

Both algorithms are available on every route request via the `algorithm` parameter.
For the route from Chacabuco & Las Heras to Avenida Falucho & Avenida Brasil, both return the identical optimal path: 4 street segments, 283.5 seconds, but through a fundamentally different process:

Dijkstra explored **1,392 nodes** across the graph before finding the solution, expanding outward in all directions with no sense of where the destination is. A* visited only **145 nodes** — a 90% reduction — by using the straight-line Haversine distance to the destination as a heuristic to prioritize nodes that are geographically closer to the goal.

The result is always the same optimal route. The difference is how much of the graph each algorithm had to examine to find it.

---

## Running locally

### Requirements
- Docker and Docker Compose

### Start

```bash
git clone https://github.com/alvarezzramiro/Geolocation-API
cd Geolocation-API
docker compose up --build
```

Docker will:
1. Build the Go binary
2. Start Neo4j and Redis
3. Wait for both to be healthy
4. Start the API — which downloads the Tandil street map from OpenStreetMap and loads it into Neo4j on first run

The API is available at `http://localhost:8080`.

On subsequent runs the seed is skipped — data persists in Docker volumes.

### Development (without Docker for the app)

```bash
# Start infrastructure only
docker compose up neo4j redis -d

# Run the app with hot reload
go install github.com/air-verse/air@latest
air
```

### Reset map data

To force a fresh download from OpenStreetMap:

```bash
docker compose down -v   # removes volumes
docker compose up
```

---

## Running tests

```bash
go test ./...
```

Tests cover the pathfinding algorithms, path compression, intersection parsing, and OSM data processing, without requiring a running database.

```bash
# Verbose output per package
go test ./graph/... -v
go test ./api/... -v
go test ./seed/... -v
```

---

## Configuration

All settings are read from environment variables with sensible defaults for local development.

---

## Intersection search

The `from` and `to` parameters accept street names separated by `/`, or `-`. Search is case-insensitive and accent-insensitive: `maipu/san martin` finds `Maipú & San Martín`.

Street names are stored with a normalized version (`normalized_name`) on every `ROAD` relationship in Neo4j, built at seed time. Queries match against the normalized version to avoid accent sensitivity at query time.

---

## Data source

Street data is downloaded from OpenStreetMap via the Overpass API using this bounding box for central Tandil, Argentina:

```
(-37.340, -59.155, -37.300, -59.110)
```

The loaded graph covers 3,242 intersections and 4,269 street segments including one-way streets, speed limits, and named intersections.
