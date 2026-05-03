// api/handlers.go
package api

import (
	"encoding/json"
	"log"
	"math"
	"net/http"

	"github.com/alvarezzramiro/street-router/db"
	"github.com/alvarezzramiro/street-router/graph"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
)

// Handler agrupa las dependencias que necesitan los endpoints.
// En vez de variables globales, las pasamos explícitamente.
type Handler struct {
	graph graph.Graph
	neo4j neo4j.DriverWithContext
	redis *redis.Client
}

// NewHandler construye el Handler con sus dependencias.
func NewHandler(g graph.Graph, neo4jDriver neo4j.DriverWithContext, rdb *redis.Client) *Handler {
	return &Handler{graph: g, neo4j: neo4jDriver, redis: rdb}
}

// writeJSON serializa v como JSON y lo escribe en la respuesta.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError escribe un error como JSON con el formato {"error": "mensaje"}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// GetRoute maneja GET /route?from=n1&to=n8
func (h *Handler) GetRoute(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	// Validar que los parámetros estén presentes
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "los parámetros 'from' y 'to' son requeridos")
		return
	}

	// Validar que los nodos existen en el grafo
	if _, ok := h.graph[from]; !ok {
		writeError(w, http.StatusNotFound, "nodo origen no encontrado: "+from)
		return
	}
	if _, ok := h.graph[to]; !ok {
		writeError(w, http.StatusNotFound, "nodo destino no encontrado: "+to)
		return
	}

	ctx := r.Context()

	// 1. Intentar leer desde caché
	if cached, ok := db.GetRoute(ctx, h.redis, from, to); ok {
		log.Printf("cache HIT  %s -> %s", from, to)
		writeJSON(w, http.StatusOK, map[string]any{
			"source": "cache",
			"result": cached,
		})
		return
	}

	// 2. Cache miss — calcular con Dijkstra
	log.Printf("cache MISS %s -> %s", from, to)
	result, err := graph.FindRoute(h.graph, from, to)
	if err == nil {
		result.Steps = graph.CompressSteps(result.Steps)
	}
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// 3. Guardar en caché para la próxima vez
	if err := db.SetRoute(ctx, h.redis, from, to, result); err != nil {
		// Un error de caché no es fatal — respondemos igual
		log.Printf("advertencia: no se pudo cachear ruta: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source": "computed",
		"result": result,
	})
}

// GetNodes maneja GET /nodes — devuelve todos los nodos del grafo.
// Útil para saber qué IDs usar en /route.
func (h *Handler) GetNodes(w http.ResponseWriter, r *http.Request) {
	session := h.neo4j.NewSession(r.Context(), neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(r.Context())

	result, err := session.Run(r.Context(),
		`MATCH (n:Intersection)
		 RETURN n.id AS id, n.name AS name, n.type AS type,
		        n.lat AS lat, n.lon AS lon
		 ORDER BY n.id`,
		nil,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error consultando nodos")
		return
	}

	type Node struct {
		ID   string  `json:"id"`
		Name string  `json:"name"`
		Type string  `json:"type"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
	}

	nodes := []Node{}
	for result.Next(r.Context()) {
		rec := result.Record()
		id, _ := rec.Get("id")
		name, _ := rec.Get("name")
		typ, _ := rec.Get("type")
		lat, _ := rec.Get("lat")
		lon, _ := rec.Get("lon")

		nodes = append(nodes, Node{
			ID:   id.(string),
			Name: name.(string),
			Type: typ.(string),
			Lat:  lat.(float64),
			Lon:  lon.(float64),
		})
	}

	writeJSON(w, http.StatusOK, nodes)
}

// api/handlers.go — agregar debajo de GetRouteByName (o reemplazarlo)

// GetRouteByIntersection maneja:
// GET /route/by-intersection?from=San+Martín+y+Pinto&to=9+de+Julio+y+Rodriguez
func (h *Handler) GetRouteByIntersection(w http.ResponseWriter, r *http.Request) {
	fromParam := r.URL.Query().Get("from")
	toParam := r.URL.Query().Get("to")

	if fromParam == "" || toParam == "" {
		writeError(w, http.StatusBadRequest,
			"parámetros requeridos: from='Calle A y Calle B'&to='Calle C y Calle D'")
		return
	}

	// Parsear las intersecciones
	fromStreet1, fromStreet2, err := parseIntersection(fromParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, "parámetro 'from': "+err.Error())
		return
	}

	toStreet1, toStreet2, err := parseIntersection(toParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, "parámetro 'to': "+err.Error())
		return
	}

	ctx := r.Context()

	// Buscar los nodos de cada intersección en Neo4j
	fromNodes, err := db.NodeByIntersection(ctx, h.neo4j, fromStreet1, fromStreet2)
	if err != nil {
		writeError(w, http.StatusNotFound, "origen: "+err.Error())
		return
	}

	toNodes, err := db.NodeByIntersection(ctx, h.neo4j, toStreet1, toStreet2)
	if err != nil {
		writeError(w, http.StatusNotFound, "destino: "+err.Error())
		return
	}

	// Si hay múltiples candidatos (calle que se cruza dos veces),
	// tomamos el primero — en práctica urbana casi nunca hay más de uno.
	fromID := fromNodes[0].ID
	toID := toNodes[0].ID

	// Caché
	if cached, ok := db.GetRoute(ctx, h.redis, fromID, toID); ok {
		log.Printf("cache HIT  %s -> %s", fromID, toID)
		writeJSON(w, http.StatusOK, map[string]any{
			"source": "cache",
			"resolved": map[string]string{
				"from": fromNodes[0].Name,
				"to":   toNodes[0].Name,
			},
			"result": cached,
		})
		return
	}

	// Dijkstra
	log.Printf("cache MISS %s -> %s", fromID, toID)

	result, err := graph.FindRoute(h.graph, fromID, toID)
	if err == nil {
		result.Steps = graph.CompressSteps(result.Steps)
	}

	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if err := db.SetRoute(ctx, h.redis, fromID, toID, result); err != nil {
		log.Printf("advertencia: no se pudo cachear: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source": "computed",
		"resolved": map[string]string{
			"from": fromNodes[0].Name,
			"to":   toNodes[0].Name,
		},
		"result": map[string]any{
			"Steps":     graph.CompressSteps(result.Steps),
			"TotalSecs": math.Round(result.TotalSecs*10) / 10,
			"TotalMins": math.Round(result.TotalSecs/60*10) / 10,
		},
	})
}
