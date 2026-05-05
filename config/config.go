// config/config.go
package config

import "os"

// Agrupa toda la configuracion de la app.
// cada campo se lee desde variables de entorno,
// con un valor por defecto se la variable no existe.

type Config struct {
	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string
	RedisAddr     string
	ServerAddr    string
}

// Load lee el entorno y devuelve una Config lista para usar.

func Load() Config {
	return Config{
		Neo4jURI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "password123"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		ServerAddr:    getEnv("SERVER_ADDR", ":8080"),
	}
}

// getEnv devuleve el valor de una variable de entorno,
// o el fallback si la variable no esta definida.

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
