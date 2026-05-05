package graph

import (
	"container/heap"
	"context"
	"fmt"
	"math"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Edge es una arista del grafo con el respectivo peso
type Edge struct {
	To     string
	Weight float64
	Street string
}

// Grafo completo en memoria. Cada clave es un nodo.
type Graph map[string][]Edge

// Tramo
type RouteStep struct {
	From   string
	To     string
	Street string
}

type Result struct {
	Steps     []RouteStep
	TotalSecs float64 // tiempo total en segundos
}

// Item: elemento de la cola de prioridad
type Item struct {
	id   string  // id del nodo
	cost float64 // costo acumulado hasta el nodo
}

type PriorityQueue []*Item

type NodeNames map[string]string

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].cost < pq[j].cost }
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *PriorityQueue) Push(x any)        { *pq = append(*pq, x.(*Item)) }
func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}

// Se leen todas las relaciones road de neo4j y construye el grafo en memoria.
func LoadGraph(ctx context.Context, driver neo4j.DriverWithContext) (Graph, NodeNames, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	// Query 1: relaciones para el grafo
	result, err := session.Run(ctx,
		`MATCH (a:Intersection)-[r:ROAD]->(b:Intersection)
		 RETURN a.id AS from, b.id AS to, r.weight AS weight, r.name AS street`,
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error cargando grafo: %w", err)
	}

	g := make(Graph)
	for result.Next(ctx) {
		record := result.Record()
		from, _ := record.Get("from")
		to, _ := record.Get("to")
		weight, _ := record.Get("weight")
		street, _ := record.Get("street")
		fromID := from.(string)
		g[fromID] = append(g[fromID], Edge{
			To:     to.(string),
			Weight: weight.(float64),
			Street: street.(string),
		})
	}
	if err := result.Err(); err != nil {
		return nil, nil, fmt.Errorf("error leyendo relaciones: %w", err)
	}

	// Query 2: nombres de nodos
	nameResult, err := session.Run(ctx,
		`MATCH (n:Intersection) RETURN n.id AS id, n.name AS name`,
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error cargando nombres: %w", err)
	}

	names := make(NodeNames)
	for nameResult.Next(ctx) {
		rec := nameResult.Record()
		id, _ := rec.Get("id")
		name, _ := rec.Get("name")
		names[id.(string)] = name.(string)
	}
	if err := nameResult.Err(); err != nil {
		return nil, nil, fmt.Errorf("error leyendo nombres: %w", err)
	}

	return g, names, nil
}

// Dijkstra. Devuelve los pasos del camino y el tiempo total en segundos.
func FindRoute(g Graph, start, end string) (Result, error) {

	// dist: menor costo conocido para llegar a cada nodo.
	// inicialmente inf para todos excepto inicial.
	dist := make(map[string]float64)
	for id := range g {
		dist[id] = math.Inf(1)
	}
	dist[start] = 0

	// prev: para cada nodo, desde que nodo y po que calle se llega
	type Arrival struct {
		from   string
		street string
	}
	prev := make(map[string]Arrival)

	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &Item{id: start, cost: 0})

	for pq.Len() > 0 {
		curr := heap.Pop(pq).(*Item)

		if curr.id == end {
			break
		}

		if curr.cost > dist[curr.id] {
			continue
		}

		for _, edge := range g[curr.id] {
			newCost := dist[curr.id] + edge.Weight

			if newCost < dist[edge.To] {
				dist[edge.To] = newCost
				prev[edge.To] = Arrival{from: curr.id, street: edge.Street}
				heap.Push(pq, &Item{id: edge.To, cost: newCost})
			}
		}
	}

	// Si dist[end] sigue siendo infinito, no hay camino
	if math.IsInf(dist[end], 1) {
		return Result{}, fmt.Errorf("no hay ruta entre %s y %s", start, end)
	}

	// Reconstruir camino de atras para adelante.
	steps := []RouteStep{}
	current := end
	for current != start {
		arrival := prev[current]
		steps = append([]RouteStep{{
			From:   arrival.from,
			To:     current,
			Street: arrival.street,
		}}, steps...)
		current = arrival.from
	}

	return Result{
		Steps:     steps,
		TotalSecs: dist[end],
	}, nil
}

// CompressSteps agrupa los pasos consecutivos de la misma calle
// en un solo tramo. En vez de 7 pasos por "General Pinto",
// devuelve un solo paso "General Pinto" de principio a fin.
func CompressSteps(steps []RouteStep) []RouteStep {
	if len(steps) == 0 {
		return steps
	}

	compressed := []RouteStep{steps[0]}

	for i := 1; i < len(steps); i++ {
		last := &compressed[len(compressed)-1]
		curr := steps[i]

		// Si la calle es la misma que el tramo anterior,
		// simplemente extendemos el destino del último tramo.
		if curr.Street == last.Street {
			last.To = curr.To
		} else {
			compressed = append(compressed, curr)
		}
	}

	return compressed
}
