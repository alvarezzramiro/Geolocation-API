// graph/astar_test.go
package graph

import (
	"testing"
)

func TestAStar_optimalRoute(t *testing.T) {
	g := testGraph()
	coords := testCoords()
	router := &AStarRouter{}

	result, err := router.FindRoute(g, coords, "A", "C")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if result.TotalSecs != 5 {
		t.Errorf("TotalSecs incorrecto: got %.1f, want 5.0", result.TotalSecs)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("esperaba 2 steps, got %d", len(result.Steps))
	}
}

func TestAStar_noRoute(t *testing.T) {
	g := testGraph()
	coords := testCoords()
	router := &AStarRouter{}

	_, err := router.FindRoute(g, coords, "C", "A")
	if err == nil {
		t.Error("esperaba error para ruta inexistente, got nil")
	}
}

func TestAStar_sameResultAsDijkstra(t *testing.T) {
	// Ambos algoritmos deben encontrar la misma ruta óptima
	g := testGraph()
	coords := testCoords()

	dijkstra := &DijkstraRouter{}
	astar := &AStarRouter{}

	rD, err := dijkstra.FindRoute(g, coords, "A", "C")
	if err != nil {
		t.Fatalf("dijkstra error: %v", err)
	}

	rA, err := astar.FindRoute(g, coords, "A", "C")
	if err != nil {
		t.Fatalf("astar error: %v", err)
	}

	if rD.TotalSecs != rA.TotalSecs {
		t.Errorf("TotalSecs difiere: dijkstra=%.1f astar=%.1f", rD.TotalSecs, rA.TotalSecs)
	}
	if len(rD.Steps) != len(rA.Steps) {
		t.Errorf("cantidad de steps difiere: dijkstra=%d astar=%d", len(rD.Steps), len(rA.Steps))
	}
}

func TestAStar_fewerNodesVisited(t *testing.T) {
	// En un grafo más grande, A* debería visitar menos nodos que Dijkstra.
	// Construimos un grafo lineal de 10 nodos donde la heurística
	// guía claramente hacia el destino.
	g := make(Graph)
	coords := make(NodeCoords)

	nodes := []string{"n0", "n1", "n2", "n3", "n4", "n5", "n6", "n7", "n8", "n9"}
	for i, id := range nodes {
		coords[id] = [2]float64{-37.30 - float64(i)*0.01, -59.10}
		if i < len(nodes)-1 {
			g[id] = []Edge{{To: nodes[i+1], Weight: 10, Street: "Calle"}}
		} else {
			g[id] = []Edge{}
		}
	}

	dijkstra := &DijkstraRouter{}
	astar := &AStarRouter{}

	rD, _ := dijkstra.FindRoute(g, coords, "n0", "n9")
	rA, _ := astar.FindRoute(g, coords, "n0", "n9")

	if rA.NodesVisited > rD.NodesVisited {
		t.Errorf("A* visitó más nodos que Dijkstra: astar=%d dijkstra=%d",
			rA.NodesVisited, rD.NodesVisited)
	}
}
