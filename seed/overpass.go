// seed/overpass.go
package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
)

type Intersection struct {
	ID   string
	Name string
	Lat  float64
	Lon  float64
	Type string // "intersection", "poi", "dead_end"
}

type Road struct {
	FromID   string
	ToID     string
	Name     string
	Distance float64 // metros
	Speed    float64 // km/h permitidos
	Oneway   bool
}

// --- Estructuras que mapean la respuesta JSON de Overpass ---

// overpassResponse es el envelope raíz que devuelve la API.
type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

// overpassElement puede ser un "node" (intersección) o un "way" (calle).
// Overpass mezcla ambos en el mismo array, diferenciados por el campo Type.
type overpassElement struct {
	Type  string            `json:"type"`
	ID    int64             `json:"id"`
	Lat   float64           `json:"lat"`   // solo en nodes
	Lon   float64           `json:"lon"`   // solo en nodes
	Nodes []int64           `json:"nodes"` // solo en ways: lista de node IDs que forman la calle
	Tags  map[string]string `json:"tags"`  // metadatos: nombre, sentido, velocidad, etc.
}

// osmNode es un nodo procesado, listo para insertar en Neo4j.
type osmNode struct {
	ID  int64
	Lat float64
	Lon float64
}

// La query Overpass QL que le manda al servidor.
// Explicación línea por línea:
//
//	[out:json]       — respuesta en JSON (también existe XML)
//	[timeout:30]     — máximo 30 segundos de procesamiento en el servidor
//	way["highway"]   — calles (ways) que tengan la tag "highway"
//	["name"]         — que además tengan nombre (filtra caminos sin nombre)
//	(bbox)           — dentro del bounding box del centro de Tandil
//	out body         — devuelve los ways con sus tags
//	>                — también devuelve los nodes que componen esos ways
//	out skel qt      — los nodes solo con geometría (lat/lon), sin tags extra
const overpassQuery = `
[out:json][timeout:30];
(
  way["highway"]["name"](-37.340,-59.155,-37.300,-59.110);
);
out body;
>;
out skel qt;
`

// FetchFromOverpass descarga el mapa de Tandil desde OpenStreetMap
// y lo convierte a las estructuras Intersection y Road que usa el seed.
func FetchFromOverpass(ctx context.Context) ([]Intersection, []Road, error) {
	params := url.Values{}
	params.Set("data", overpassQuery)

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		"https://overpass-api.de/api/interpreter",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error creando request: %w", err)
	}

	// Con el body como form, el Content-Type correcto es este.
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Geolocation-API/1.0 (portfolio project)")
	req.Header.Set("Accept", "*/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error llamando a overpass: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("overpass respondió %d: %s", resp.StatusCode, string(body))
	}

	var result overpassResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	return parseElements(result.Elements)
}

// parseElements separa los elements en nodes y ways,
// y construye las listas de Intersection y Road.
func parseElements(elements []overpassElement) ([]Intersection, []Road, error) {
	// Paso 1: indexar nodes por ID
	nodeIndex := make(map[int64]osmNode)
	for _, el := range elements {
		if el.Type == "node" {
			nodeIndex[el.ID] = osmNode{ID: el.ID, Lat: el.Lat, Lon: el.Lon}
		}
	}

	// Paso 2: registrar a qué calles pertenece cada node
	// Usamos map[string]bool como set para deduplicar en O(1)
	nodeStreetSets := make(map[int64]map[string]bool)
	for _, el := range elements {
		if el.Type != "way" {
			continue
		}
		name := el.Tags["name"]
		if name == "" {
			continue
		}
		for _, nodeID := range el.Nodes {
			if nodeStreetSets[nodeID] == nil {
				nodeStreetSets[nodeID] = make(map[string]bool)
			}
			nodeStreetSets[nodeID][name] = true
		}
	}

	// Convertir sets a slices para buildName
	nodeStreets := make(map[int64][]string)
	for nodeID, streetSet := range nodeStreetSets {
		for name := range streetSet {
			nodeStreets[nodeID] = append(nodeStreets[nodeID], name)
		}
	}

	// Paso 3: construir intersecciones y roads
	intersectionsSeen := make(map[int64]bool)
	var intersections []Intersection
	var roads []Road

	for _, el := range elements {
		if el.Type != "way" {
			continue
		}

		name := el.Tags["name"]
		if name == "" {
			name = "sin nombre"
		}
		speed := parseSpeed(el.Tags["maxspeed"])
		oneway := el.Tags["oneway"] == "yes"

		for i := 0; i < len(el.Nodes)-1; i++ {
			fromID := el.Nodes[i]
			toID := el.Nodes[i+1]

			fromNode, okF := nodeIndex[fromID]
			toNode, okT := nodeIndex[toID]
			if !okF || !okT {
				continue
			}

			if !intersectionsSeen[fromID] {
				intersectionsSeen[fromID] = true
				intersections = append(intersections, Intersection{
					ID:   fmt.Sprintf("osm_%d", fromID),
					Name: buildName(fromID, nodeStreets),
					Lat:  fromNode.Lat,
					Lon:  fromNode.Lon,
					Type: intersectionType(fromID, nodeStreets),
				})
			}

			if !intersectionsSeen[toID] {
				intersectionsSeen[toID] = true
				intersections = append(intersections, Intersection{
					ID:   fmt.Sprintf("osm_%d", toID),
					Name: buildName(toID, nodeStreets),
					Lat:  toNode.Lat,
					Lon:  toNode.Lon,
					Type: intersectionType(toID, nodeStreets),
				})
			}

			dist := haversineMeters(fromNode.Lat, fromNode.Lon, toNode.Lat, toNode.Lon)
			roads = append(roads, Road{
				FromID:   fmt.Sprintf("osm_%d", fromID),
				ToID:     fmt.Sprintf("osm_%d", toID),
				Name:     name,
				Distance: dist,
				Speed:    speed,
				Oneway:   oneway,
			})
		}
	}

	return intersections, roads, nil
}

// buildName construye el nombre legible de una intersección.
// Si el nodo pertenece a dos calles → "Calle A y Calle B"
// Si pertenece a una sola → el nombre de esa calle
// Si no tiene calles → "nodo {id}"
func buildName(nodeID int64, nodeStreets map[int64][]string) string {
	streets := nodeStreets[nodeID]
	switch len(streets) {
	case 0:
		return fmt.Sprintf("nodo %d", nodeID)
	case 1:
		return streets[0]
	default:
		return streets[0] + " & " + streets[1]
	}
}

// intersectionType determina el tipo según cuántas calles pasan por el nodo.
func intersectionType(nodeID int64, nodeStreets map[int64][]string) string {
	if len(nodeStreets[nodeID]) >= 2 {
		return "intersection"
	}
	return "dead_end"
}

// parseSpeed convierte el string "maxspeed" de OSM a float64.
// OSM puede tener valores como "50", "30 mph", "walk", o estar vacío.
func parseSpeed(s string) float64 {
	if s == "" || s == "walk" || s == "living_street" {
		return 20.0
	}
	// Extraemos solo los dígitos del principio
	var speed float64
	fmt.Sscanf(s, "%f", &speed)
	if speed <= 0 {
		return 40.0
	}
	// Si dice "mph", convertimos a km/h
	if strings.Contains(s, "mph") {
		speed *= 1.60934
	}
	return speed
}

// haversineMeters calcula la distancia en metros entre dos puntos GPS.
// Es la distancia real sobre la superficie de la Tierra,
// no la distancia euclidiana en un plano flat.
func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0 // radio de la Tierra en metros
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
