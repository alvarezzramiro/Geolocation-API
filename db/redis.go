// db/redis.go
package db

import (
	"context"
	"fmt"
	"log"

	"github.com/alvarezzramiro/Geolocation-API/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient crea y verifica la conexión con Redis.
func NewRedisClient(cfg config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr, // corregido
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis no disponible: %w", err)
	}

	log.Println("Conectado a Redis")
	return rdb, nil
}
