package graph

import (
	"container/heap"
	"fmt"
	"math"
)

// DijkstraRouter implementa la interfaz Router usando el algoritmo de Dijkstra.
// Explora todos los nodos en orden de menor costo acumulado —
// garantiza la ruta óptima pero visita más nodos que A*.
type DijkstraRouter struct{}

func (r *DijkstraRouter) FindRoute(g Graph, _ NodeCoords, start, end string) (Result, error) {
	return dijkstra(g, start, end)
}

// Elemento de la cola de prioridad
type dijkstraItem struct {
	id   string  // id del nodo
	cost float64 // costo acumulado hasta el nodo
}

type dijkstraPQ []*dijkstraItem

func (pq dijkstraPQ) Len() int           { return len(pq) }
func (pq dijkstraPQ) Less(i, j int) bool { return pq[i].cost < pq[j].cost }
func (pq dijkstraPQ) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *dijkstraPQ) Push(x any)        { *pq = append(*pq, x.(*dijkstraItem)) }
func (pq *dijkstraPQ) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}

// Dijkstra. Devuelve los pasos del camino y el tiempo total en segundos.
func dijkstra(g Graph, start, end string) (Result, error) {

	// dist: menor costo conocido para llegar a cada nodo.
	// inicialmente inf para todos excepto inicial.
	dist := make(map[string]float64)
	for id := range g {
		dist[id] = math.Inf(1)
	}
	dist[start] = 0

	prev := make(map[string]arrival)
	nodesVisited := 0

	pq := &dijkstraPQ{}
	heap.Init(pq)
	heap.Push(pq, &dijkstraItem{id: start, cost: 0})

	for pq.Len() > 0 {
		curr := heap.Pop(pq).(*dijkstraItem)

		if curr.id == end {
			break
		}

		if curr.cost > dist[curr.id] {
			continue
		}

		nodesVisited++

		for _, edge := range g[curr.id] {
			newCost := dist[curr.id] + edge.Weight

			if newCost < dist[edge.To] {
				dist[edge.To] = newCost
				prev[edge.To] = arrival{from: curr.id, street: edge.Street}
				heap.Push(pq, &dijkstraItem{id: edge.To, cost: newCost})
			}
		}
	}

	// Si dist[end] sigue siendo infinito, no hay camino
	if math.IsInf(dist[end], 1) {
		return Result{}, fmt.Errorf("no hay ruta entre %s y %s", start, end)
	}

	result := buildResult(prev, dist[end], start, end)
	result.NodesVisited = nodesVisited

	return result, nil
}
