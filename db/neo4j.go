// db/neo4j.go
package db

import (
	"context"
	"fmt"
	"log"

	"github.com/alvarezzramiro/Geolocation-API/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NewNeo4jDriver crea y verifica la conexión con Neo4j.
func NewNeo4jDriver(cfg config.Config) (neo4j.DriverWithContext, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4jURI,
		neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPassword, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("error creando driver neo4j: %w", err)
	}

	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		return nil, fmt.Errorf("neo4j no disponible: %w", err)
	}

	log.Println("Conectado a Neo4j")
	return driver, nil
}
