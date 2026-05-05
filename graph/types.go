// graph/types.go
package graph

// Edge representa una arista del grafo —
// una conexión desde un nodo hacia otro con su costo y nombre de calle.
type Edge struct {
	To     string
	Weight float64
	Street string
}

// Graph es el grafo completo en memoria.
// Clave: ID del nodo. Valor: lista de aristas que salen de ese nodo.
type Graph map[string][]Edge

// NodeCoords almacena las coordenadas GPS de cada nodo.
// Necesario para la heurística de A*.
type NodeCoords map[string][2]float64 // [lat, lon]

// NodeNames almacena el nombre legible de cada nodo.
// Usado por los handlers para enriquecer la respuesta.
type NodeNames map[string]string

// RouteStep es un tramo del camino encontrado.
type RouteStep struct {
	From   string
	To     string
	Street string
}

// Result es la respuesta de cualquier algoritmo de ruteo.
type Result struct {
	Steps        []RouteStep
	TotalSecs    float64
	NodesVisited int
}

// arrival registra desde qué nodo y por qué calle llegamos a un nodo.
// Usado internamente por Dijkstra y A* para reconstruir el camino.
type arrival struct {
	from   string
	street string
}
