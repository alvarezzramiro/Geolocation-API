// graph/dijkstra_test.go
package graph

import (
	"testing"
)

// testGraph construye un grafo simple para los tests:
// Dijkstra y A* deben elegir A->B->C.
func testGraph() Graph {
	return Graph{
		"A": {{To: "B", Weight: 2, Street: "Calle 1"}, {To: "C", Weight: 10, Street: "Calle 2"}},
		"B": {{To: "C", Weight: 3, Street: "Calle 1"}},
		"C": {},
	}
}

func testCoords() NodeCoords {
	return NodeCoords{
		"A": {-37.320000, -59.130000},
		"B": {-37.320001, -59.130001},
		"C": {-37.320002, -59.130002},
	}
}

func TestDijkstra_optimalRoute(t *testing.T) {
	g := testGraph()
	coords := testCoords()
	router := &DijkstraRouter{}

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
	if result.Steps[0].Street != "Calle 1" {
		t.Errorf("street incorrecto en step 0: got %q", result.Steps[0].Street)
	}
}

func TestDijkstra_noRoute(t *testing.T) {
	g := testGraph()
	coords := testCoords()
	router := &DijkstraRouter{}

	// C no tiene aristas salientes — no hay ruta de C a A
	_, err := router.FindRoute(g, coords, "C", "A")
	if err == nil {
		t.Error("esperaba error para ruta inexistente, got nil")
	}
}

func TestDijkstra_sameNode(t *testing.T) {
	g := testGraph()
	coords := testCoords()
	router := &DijkstraRouter{}

	result, err := router.FindRoute(g, coords, "A", "A")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.TotalSecs != 0 {
		t.Errorf("TotalSecs debería ser 0, got %.1f", result.TotalSecs)
	}
	if len(result.Steps) != 0 {
		t.Errorf("Steps debería estar vacío, got %d", len(result.Steps))
	}
}

func TestDijkstra_nodesVisited(t *testing.T) {
	g := testGraph()
	coords := testCoords()
	router := &DijkstraRouter{}

	result, err := router.FindRoute(g, coords, "A", "C")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.NodesVisited == 0 {
		t.Error("NodesVisited debería ser mayor que 0")
	}
}
