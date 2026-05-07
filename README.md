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
| Map data | | |
