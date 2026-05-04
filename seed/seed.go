// seed/seed.go
package seed

import (
	"context"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Intersection struct {
	ID   string
	Name string
	Lat  float64
	Lon  float64
	Type string // "intersection", "poi", "dead_end"
}

type Road struct {
	FromID   string
	ToID     string
	Name     string
	Distance float64 // metros
	Speed    float64 // km/h permitidos
	Oneway   bool
}

// weight calcula el tiempo en segundos para recorrer un tramo.
// dist en metros / veloc en m/s
// veloc en m/s = km/h * 1000 / 3600
func weight(distance, speed float64) float64 {
	return distance / (speed * 1000 / 3600)
}

// Run ejecuta el seed completo
// Primero limpia la base, luego carga nodos y relaciones.
func Run(ctx context.Context, driver neo4j.DriverWithContext) error {
	log.Println("Descargando mapa desde OpenStreetMap...")

	intersections, roads, err := FetchFromOverpass(ctx)
	if err != nil {
		return err
	}

	log.Printf("OSM: %d nodos, %d segmentos descargados", len(intersections), len(roads))

	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	// Transacción 1: índice (separado por el error que vimos antes)
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx,
			"CREATE INDEX intersection_id IF NOT EXISTS FOR (n:Intersection) ON (n.id)",
			nil,
		)
		return nil, err
	})
	if err != nil {
		return err
	}

	// Transacción 2: datos
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {

		if _, err := tx.Run(ctx, "MATCH (n) DETACH DELETE n", nil); err != nil {
			return nil, err
		}

		for _, n := range intersections {
			_, err := tx.Run(ctx,
				`CREATE (:Intersection {
					id: $id, name: $name,
					lat: $lat, lon: $lon, type: $type
				})`,
				map[string]any{
					"id": n.ID, "name": n.Name,
					"lat": n.Lat, "lon": n.Lon, "type": n.Type,
				},
			)
			if err != nil {
				return nil, err
			}
		}

		for _, r := range roads {
			w := weight(r.Distance, r.Speed)

			_, err := tx.Run(ctx,
				`MATCH (a:Intersection {id: $from}), (b:Intersection {id: $to})
				CREATE (a)-[:ROAD {
					name:            $name,
					normalized_name: $normalized_name,
					distance:        $distance,
					speed:           $speed,
					weight:          $weight,
					oneway:          $oneway
				}]->(b)`,
				map[string]any{
					"from":            r.FromID,
					"to":              r.ToID,
					"name":            r.Name,
					"normalized_name": normalize(r.Name), // nuevo
					"distance":        r.Distance,
					"speed":           r.Speed,
					"weight":          w,
					"oneway":          r.Oneway,
				},
			)

			if err != nil {
				return nil, err
			}

			if !r.Oneway {
				_, err := tx.Run(ctx,
					`MATCH (a:Intersection {id: $from}), (b:Intersection {id: $to})
					 CREATE (b)-[:ROAD {
						name: $name, distance: $distance,
						speed: $speed, weight: $weight, oneway: $oneway
					 }]->(a)`,
					map[string]any{
						"from": r.FromID, "to": r.ToID, "name": r.Name,
						"distance": r.Distance, "speed": r.Speed,
						"weight": w, "oneway": r.Oneway,
					},
				)
				if err != nil {
					return nil, err
				}
			}
		}

		log.Printf("Seed completado: %d nodos, %d segmentos en Neo4j",
			len(intersections), len(roads))
		return nil, nil
	})

	return err
}
