// repository/streets.go
package repository

import (
	"context"
	"fmt"

	"github.com/alvarezzramiro/street-router/internal/text"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// StreetNode representa una intersección del mapa con sus coordenadas.
type StreetNode struct {
	ID   string
	Name string
	Lat  float64
	Lon  float64
}

// NodeByIntersection encuentra el nodo que pertenece simultáneamente
// a las dos calles dadas. Usa búsqueda normalizada — insensible a tildes.
func NodeByIntersection(ctx context.Context, driver neo4j.DriverWithContext, street1, street2 string) ([]StreetNode, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	norm1 := text.Normalize(street1)
	norm2 := text.Normalize(street2)

	result, err := session.Run(ctx,
		`MATCH (n:Intersection)-[r1:ROAD]->()
		 MATCH (n:Intersection)-[r2:ROAD]->()
		 WHERE r1.normalized_name CONTAINS $street1
		   AND r2.normalized_name CONTAINS $street2
		   AND r1.name <> r2.name
		 RETURN DISTINCT n.id AS id, n.name AS name,
		        n.lat AS lat, n.lon AS lon`,
		map[string]any{
			"street1": norm1,
			"street2": norm2,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error buscando intersección: %w", err)
	}

	var nodes []StreetNode
	for result.Next(ctx) {
		rec := result.Record()
		id, _ := rec.Get("id")
		name, _ := rec.Get("name")
		lat, _ := rec.Get("lat")
		lon, _ := rec.Get("lon")
		nodes = append(nodes, StreetNode{
			ID:   id.(string),
			Name: name.(string),
			Lat:  lat.(float64),
			Lon:  lon.(float64),
		})
	}

	if err := result.Err(); err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no se encontró la intersección entre %q y %q", street1, street2)
	}

	return nodes, nil
}
