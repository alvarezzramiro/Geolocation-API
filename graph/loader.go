// graph/loader.go
package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// LoadGraph lee el grafo, los nombres y las coordenadas desde Neo4j.
func LoadGraph(ctx context.Context, driver neo4j.DriverWithContext) (Graph, NodeNames, NodeCoords, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	// Query 1: aristas
	result, err := session.Run(ctx,
		`MATCH (a:Intersection)-[r:ROAD]->(b:Intersection)
		 RETURN a.id AS from, b.id AS to, r.weight AS weight, r.name AS street`,
		nil,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error cargando grafo: %w", err)
	}

	g := make(Graph)
	for result.Next(ctx) {
		rec := result.Record()
		from, _ := rec.Get("from")
		to, _ := rec.Get("to")
		weight, _ := rec.Get("weight")
		street, _ := rec.Get("street")
		fromID := from.(string)
		g[fromID] = append(g[fromID], Edge{
			To:     to.(string),
			Weight: weight.(float64),
			Street: street.(string),
		})
	}
	if err := result.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("error leyendo aristas: %w", err)
	}

	// Query 2: nombres y coordenadas
	nodeResult, err := session.Run(ctx,
		`MATCH (n:Intersection)
		 RETURN n.id AS id, n.name AS name, n.lat AS lat, n.lon AS lon`,
		nil,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error cargando nodos: %w", err)
	}

	names := make(NodeNames)
	coords := make(NodeCoords)
	for nodeResult.Next(ctx) {
		rec := nodeResult.Record()
		id, _ := rec.Get("id")
		name, _ := rec.Get("name")
		lat, _ := rec.Get("lat")
		lon, _ := rec.Get("lon")
		idStr := id.(string)
		names[idStr] = name.(string)
		coords[idStr] = [2]float64{lat.(float64), lon.(float64)}
	}
	if err := nodeResult.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("error leyendo nodos: %w", err)
	}

	return g, names, coords, nil
}
