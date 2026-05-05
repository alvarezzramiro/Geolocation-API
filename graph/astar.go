// graph/astar.go
package graph

import (
	"container/heap"
	"fmt"
	"math"
)

// AStarRouter implementa la interfaz Router usando el algoritmo A*.
// Usa una heurística geográfica (distancia Haversine al destino)
// para guiar la búsqueda — visita menos nodos que Dijkstra
// en grafos con coordenadas GPS.
type AStarRouter struct{}

func (r *AStarRouter) FindRoute(g Graph, coords NodeCoords, start, end string) (Result, error) {
	return astar(g, coords, start, end)
}

// --- Priority Queue ---

type astarItem struct {
	id string
	f  float64 // f = g + h  (costo acumulado + heurística)
}

type astarPQ []*astarItem

func (pq astarPQ) Len() int           { return len(pq) }
func (pq astarPQ) Less(i, j int) bool { return pq[i].f < pq[j].f }
func (pq astarPQ) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *astarPQ) Push(x any)        { *pq = append(*pq, x.(*astarItem)) }
func (pq *astarPQ) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}

// --- Heurística ---

// heuristic estima el costo restante desde un nodo hasta el destino.
// Usa la distancia Haversine dividida por la velocidad máxima asumida
// para obtener una estimación en segundos — la misma unidad que los weights.
// Es admisible (nunca sobreestima) porque asume movimiento en línea recta
// a la velocidad máxima posible.
func heuristic(coords NodeCoords, from, to string) float64 {
	f, okF := coords[from]
	t, okT := coords[to]
	if !okF || !okT {
		return 0 // si no hay coords, degradamos a Dijkstra
	}

	const maxSpeedMS = 50.0 / 3.6 // 50 km/h en m/s — velocidad máxima urbana
	dist := haversine(f[0], f[1], t[0], t[1])
	return dist / maxSpeedMS
}

// haversine calcula la distancia en metros entre dos puntos GPS.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// --- Algoritmo ---

func astar(g Graph, coords NodeCoords, start, end string) (Result, error) {
	// g: costo acumulado real desde start hasta cada nodo
	gCost := make(map[string]float64)
	for id := range g {
		gCost[id] = math.Inf(1)
	}
	gCost[start] = 0

	prev := make(map[string]arrival)
	nodesVisited := 0

	pq := &astarPQ{}
	heap.Init(pq)
	heap.Push(pq, &astarItem{
		id: start,
		f:  heuristic(coords, start, end), // f inicial = 0 + h
	})

	for pq.Len() > 0 {
		curr := heap.Pop(pq).(*astarItem)

		if curr.id == end {
			break
		}

		// Salteamos items obsoletos — mismo patrón que Dijkstra
		if curr.f > gCost[curr.id]+heuristic(coords, curr.id, end) {
			continue
		}

		nodesVisited++

		for _, edge := range g[curr.id] {
			newG := gCost[curr.id] + edge.Weight

			if newG < gCost[edge.To] {
				gCost[edge.To] = newG
				prev[edge.To] = arrival{from: curr.id, street: edge.Street}

				// f = costo real acumulado + estimación al destino
				f := newG + heuristic(coords, edge.To, end)
				heap.Push(pq, &astarItem{id: edge.To, f: f})
			}
		}
	}

	if math.IsInf(gCost[end], 1) {
		return Result{}, fmt.Errorf("no hay ruta entre %s y %s", start, end)
	}

	result := buildResult(prev, gCost[end], start, end)
	result.NodesVisited = nodesVisited
	return result, nil
}
