// seed/seed.go
package seed

import (
	"context"
	"fmt"
	"log"

	"github.com/alvarezzramiro/Geolocation-API/internal/text"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// weight calcula el tiempo en segundos para recorrer un tramo.
// dist en metros / veloc en m/s
// veloc en m/s = km/h * 1000 / 3600
func weight(distance, speed float64) float64 {
	return distance / (speed * 1000 / 3600)
}

// hasData verifica si Neo4j ya tiene intersecciones cargadas.
// Se usa para saltear el seed si los datos ya existen.
func hasData(ctx context.Context, driver neo4j.DriverWithContext) (bool, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx,
		"MATCH (n:Intersection) RETURN count(n) AS total",
		nil,
	)
	if err != nil {
		return false, fmt.Errorf("error verificando datos: %w", err)
	}

	record, err := result.Single(ctx)
	if err != nil {
		return false, fmt.Errorf("error leyendo conteo: %w", err)
	}

	total, _ := record.Get("total")
	return total.(int64) > 0, nil
}

// Run ejecuta el seed completo
// Primero limpia la base, luego carga nodos y relaciones.
func Run(ctx context.Context, driver neo4j.DriverWithContext) error {

	exists, err := hasData(ctx, driver)
	if err != nil {
		return err
	}
	if exists {
		log.Println("Datos ya cargados en Neo4j, saltando seed")
		return nil
	}

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

			// A -> B
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
					"normalized_name": text.Normalize(r.Name),
					"distance":        r.Distance,
					"speed":           r.Speed,
					"weight":          w,
					"oneway":          r.Oneway,
				},
			)
			if err != nil {
				return nil, err
			}

			// B -> A — solo si es doble mano
			if !r.Oneway {
				_, err := tx.Run(ctx,
					`MATCH (a:Intersection {id: $from}), (b:Intersection {id: $to})
					CREATE (b)-[:ROAD {
						name:            $name,
						normalized_name: $normalized_name,
						distance:        $distance,
						speed:           $speed,
						weight:          $weight,
						oneway:          $oneway
					}]->(a)`,
					map[string]any{
						"from":            r.FromID,
						"to":              r.ToID,
						"name":            r.Name,
						"normalized_name": text.Normalize(r.Name),
						"distance":        r.Distance,
						"speed":           r.Speed,
						"weight":          w,
						"oneway":          r.Oneway,
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
