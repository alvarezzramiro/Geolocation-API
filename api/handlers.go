// api/handlers.go
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"

	"github.com/alvarezzramiro/street-router/graph"
	"github.com/alvarezzramiro/street-router/repository"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	graph graph.Graph
	names graph.NodeNames
	neo4j neo4j.DriverWithContext
	redis *redis.Client
}

func NewHandler(g graph.Graph, names graph.NodeNames, neo4jDriver neo4j.DriverWithContext, rdb *redis.Client) *Handler {
	return &Handler{graph: g, names: names, neo4j: neo4jDriver, redis: rdb}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// enrichSteps reemplaza los IDs de los steps por nombres legibles
// manteniendo el ID entre paréntesis para referencia.
func (h *Handler) enrichSteps(steps []graph.RouteStep) []map[string]string {
	result := make([]map[string]string, len(steps))
	for i, step := range steps {
		result[i] = map[string]string{
			"from":   fmt.Sprintf("%s (%s)", h.names[step.From], step.From),
			"to":     fmt.Sprintf("%s (%s)", h.names[step.To], step.To),
			"street": step.Street,
		}
	}
	return result
}

// formatResult construye la respuesta final de una ruta de forma consistente.
// Usado por todos los handlers para garantizar el mismo formato.
func (h *Handler) formatResult(r graph.Result) map[string]any {
	compressed := graph.CompressSteps(r.Steps)
	return map[string]any{
		"Steps":     h.enrichSteps(compressed),
		"TotalSecs": math.Round(r.TotalSecs*10) / 10,
		"TotalMins": math.Round(r.TotalSecs/60*10) / 10,
	}
}

// GetRoute maneja GET /route?from=osm_XXX&to=osm_YYY
func (h *Handler) GetRoute(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "los parámetros 'from' y 'to' son requeridos")
		return
	}

	if _, ok := h.graph[from]; !ok {
		writeError(w, http.StatusNotFound, "nodo origen no encontrado: "+from)
		return
	}
	if _, ok := h.graph[to]; !ok {
		writeError(w, http.StatusNotFound, "nodo destino no encontrado: "+to)
		return
	}

	ctx := r.Context()

	if cached, ok := repository.GetRoute(ctx, h.redis, from, to); ok {
		log.Printf("cache HIT  %s -> %s", from, to)
		writeJSON(w, http.StatusOK, map[string]any{
			"source": "cache",
			"result": h.formatResult(cached),
		})
		return
	}

	log.Printf("cache MISS %s -> %s", from, to)
	result, err := graph.FindRoute(h.graph, from, to)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if err := repository.SetRoute(ctx, h.redis, from, to, result); err != nil {
		log.Printf("advertencia: no se pudo cachear ruta: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source": "computed",
		"result": h.formatResult(result),
	})
}

// GetNodes maneja GET /nodes
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

// GetRouteByIntersection maneja:
// GET /route/by-intersection?from=Calle A/Calle B&to=Calle C/Calle D
func (h *Handler) GetRouteByIntersection(w http.ResponseWriter, r *http.Request) {
	fromParam := r.URL.Query().Get("from")
	toParam := r.URL.Query().Get("to")

	if fromParam == "" || toParam == "" {
		writeError(w, http.StatusBadRequest,
			"parámetros requeridos: from='Calle A/Calle B'&to='Calle C/Calle D'")
		return
	}

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

	fromNodes, err := repository.NodeByIntersection(ctx, h.neo4j, fromStreet1, fromStreet2)
	if err != nil {
		writeError(w, http.StatusNotFound, "origen: "+err.Error())
		return
	}

	toNodes, err := repository.NodeByIntersection(ctx, h.neo4j, toStreet1, toStreet2)
	if err != nil {
		writeError(w, http.StatusNotFound, "destino: "+err.Error())
		return
	}

	fromID := fromNodes[0].ID
	toID := toNodes[0].ID

	if cached, ok := repository.GetRoute(ctx, h.redis, fromID, toID); ok {
		log.Printf("cache HIT  %s -> %s", fromID, toID)
		writeJSON(w, http.StatusOK, map[string]any{
			"source": "cache",
			"resolved": map[string]string{
				"from": fromNodes[0].Name,
				"to":   toNodes[0].Name,
			},
			"result": h.formatResult(cached),
		})
		return
	}

	log.Printf("cache MISS %s -> %s", fromID, toID)
	result, err := graph.FindRoute(h.graph, fromID, toID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if err := repository.SetRoute(ctx, h.redis, fromID, toID, result); err != nil {
		log.Printf("advertencia: no se pudo cachear: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source": "computed",
		"resolved": map[string]string{
			"from": fromNodes[0].Name,
			"to":   toNodes[0].Name,
		},
		"result": h.formatResult(result),
	})
}
