// repository/cache.go
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alvarezzramiro/street-router/graph"
	"github.com/redis/go-redis/v9"
)

const cacheTTL = 10 * time.Minute

func cacheKey(algorithm, from, to string) string {
	return fmt.Sprintf("route:%s:%s:%s", algorithm, from, to)
}

// GetRoute intenta leer una ruta del caché de Redis.
// Devuelve el resultado y true si existe, false si no.
func GetRoute(ctx context.Context, rdb *redis.Client, algorithm, from, to string) (graph.Result, bool) {
	val, err := rdb.Get(ctx, cacheKey(algorithm, from, to)).Result()
	if err != nil {
		return graph.Result{}, false
	}

	var result graph.Result
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return graph.Result{}, false
	}

	return result, true
}

// SetRoute guarda una ruta en Redis con TTL de 10 minutos.
func SetRoute(ctx context.Context, rdb *redis.Client, algorithm, from, to string, result graph.Result) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error serializando ruta: %w", err)
	}

	return rdb.Set(ctx, cacheKey(algorithm, from, to), data, cacheTTL).Err()
}
