// graph/path.go
package graph

// buildResult reconstruye el camino desde el mapa prev
// y devuelve un Result con los steps en orden y el tiempo total.
func buildResult(prev map[string]arrival, totalSecs float64, start, end string) Result {
	steps := []RouteStep{}
	current := end
	for current != start {
		a := prev[current]
		steps = append([]RouteStep{{
			From:   a.from,
			To:     current,
			Street: a.street,
		}}, steps...)
		current = a.from
	}
	return Result{Steps: steps, TotalSecs: totalSecs}
}

// CompressSteps agrupa pasos consecutivos de la misma calle en un solo tramo.
func CompressSteps(steps []RouteStep) []RouteStep {
	if len(steps) == 0 {
		return steps
	}
	compressed := []RouteStep{steps[0]}
	for i := 1; i < len(steps); i++ {
		last := &compressed[len(compressed)-1]
		if steps[i].Street == last.Street {
			last.To = steps[i].To
		} else {
			compressed = append(compressed, steps[i])
		}
	}
	return compressed
}
