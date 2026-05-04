// api/parse.go
package api

import (
	"fmt"
	"strings"
)

// parseIntersection parte un string como "San Martín y Pinto"
// en sus dos calles componentes.
// Acepta " y ", " & ", " / " como separadores.
func parseIntersection(s string) (street1, street2 string, err error) {
	separators := []string{" & ", "&", " / ", " - "}

	for _, sep := range separators {
		parts := strings.SplitN(strings.ToLower(s), sep, 2)
		if len(parts) == 2 {
			// Reconstruir con capitalización original buscando el separador
			idx := strings.Index(strings.ToLower(s), sep)
			street1 = strings.TrimSpace(s[:idx])
			street2 = strings.TrimSpace(s[idx+len(sep):])
			return
		}
	}

	return "", "", fmt.Errorf(
		"formato inválido: %q — usá 'Calle A / Calle B'", s)
}
