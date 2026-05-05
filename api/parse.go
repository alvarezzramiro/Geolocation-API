// api/parse.go
package api

import (
	"fmt"
	"strings"
)

// parseIntersection parte un string como "San Martín & Pinto"
// en sus dos calles componentes.
// Separadores aceptados: / & -
func parseIntersection(s string) (street1, street2 string, err error) {
	separators := []string{" & ", "&", " / ", "/", " - ", "-"}

	for _, sep := range separators {
		idx := strings.Index(strings.ToLower(s), strings.ToLower(sep))
		if idx != -1 {
			street1 = strings.TrimSpace(s[:idx])
			street2 = strings.TrimSpace(s[idx+len(sep):])
			return
		}
	}

	return "", "", fmt.Errorf(
		"formato inválido: %q — usá 'Calle A / Calle B'", s)
}
