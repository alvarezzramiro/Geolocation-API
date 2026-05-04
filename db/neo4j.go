// db/neo4j.go
package db

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/alvarezzramiro/street-router/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NewNeo4jDriver crea y verifica la conexion de neo4j.
// Devuelve el driver listo para usar, o un error si no se pudo conectar.
func NewNeo4jDriver(cfg config.Config) (neo4j.DriverWithContext, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4jURI,
		neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPassword, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("error creando driver neo4j: %w", err)
	}

	// VerifyConnectivity intenta abrir una conexion real.
	// Si Neo4j no esta leyendo, falla aca.
	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		return nil, fmt.Errorf("neo4j no disponible: %w", err)
	}

	log.Println("Conectado a Neo4j")
	return driver, nil
}

// StreetNode representa un nodo del grafo con sus coordenadas.
type StreetNode struct {
	ID   string
	Name string
	Lat  float64
	Lon  float64
}

// db/neo4j.go — agregar
func normalize(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
		"ä", "a", "ë", "e", "ï", "i", "ö", "o", "ü", "u",
		"ñ", "n",
	)
	return replacer.Replace(s)
}

func NodeByIntersection(ctx context.Context, driver neo4j.DriverWithContext, street1, street2 string) ([]StreetNode, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	// Normalizamos el input antes de mandarlo a Neo4j
	norm1 := normalize(street1)
	norm2 := normalize(street2)

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
