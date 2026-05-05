// graph/router.go
package graph

// Router es la interfaz que cualquier algoritmo de ruteo debe implementar.
// Permite intercambiar Dijkstra por A* sin cambiar el código del handler.
type Router interface {
	FindRoute(g Graph, coords NodeCoords, start, end string) (Result, error)
}

// Algorithm identifica el algoritmo de ruteo a usar.
type Algorithm string

const (
	AlgorithmDijkstra Algorithm = "dijkstra"
	AlgorithmAStar    Algorithm = "astar"
)

// NewRouter devuelve el Router correspondiente al algoritmo pedido.
// Si el algoritmo no se reconoce, usa Dijkstra por defecto.
func NewRouter(alg Algorithm) Router {
	switch alg {
	case AlgorithmAStar:
		return &AStarRouter{}
	default:
		return &DijkstraRouter{}
	}
}
