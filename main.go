// main.go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/alvarezzramiro/street-router/api"
	"github.com/alvarezzramiro/street-router/config"
	"github.com/alvarezzramiro/street-router/db"
	"github.com/alvarezzramiro/street-router/graph"
	"github.com/alvarezzramiro/street-router/seed"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	neo4jDriver, err := db.NewNeo4jDriver(cfg)
	if err != nil {
		log.Fatalf("No se pudo conectar a Neo4j: %v", err)
	}
	defer neo4jDriver.Close(ctx)

	redisClient, err := db.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("No se pudo conectar a Redis: %v", err)
	}
	defer redisClient.Close()

	if err := seed.Run(ctx, neo4jDriver); err != nil {
		log.Fatalf("Error en seed: %v", err)
	}

	g, names, coords, err := graph.LoadGraph(ctx, neo4jDriver)
	if err != nil {
		log.Fatalf("Error cargando grafo: %v", err)
	}
	log.Printf("Grafo cargado: %d nodos", len(g))

	handler := api.NewHandler(g, names, coords, neo4jDriver, redisClient)
	router := api.NewRouter(handler)

	log.Printf("Servidor escuchando en http://localhost%s", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, router); err != nil {
		log.Fatalf("Error en servidor: %v", err)
	}
}
