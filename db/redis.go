// dv/redis.go
package db

import (
	"context"
	"fmt"
	"log"

	"github.com/alvarezzramiro/street-router/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient crea y verifica la conexion con redis
func NewRedisClient(cfg config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedIsAddr,
	})

	// Ping verifica que redis responde.
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis no disponible: %w", err)
	}

	log.Println("Conectado a redis")
	return rdb, nil
}
