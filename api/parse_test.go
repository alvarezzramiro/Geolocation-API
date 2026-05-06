// api/parse_test.go
package api

import (
	"testing"
)

func TestParseIntersection_slashSeparator(t *testing.T) {
	s1, s2, err := parseIntersection("San Martín/Pinto")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if s1 != "San Martín" {
		t.Errorf("street1 incorrecto: got %q, want %q", s1, "San Martín")
	}
	if s2 != "Pinto" {
		t.Errorf("street2 incorrecto: got %q, want %q", s2, "Pinto")
	}
}

func TestParseIntersection_slashWithSpaces(t *testing.T) {
	s1, s2, err := parseIntersection("San Martín / Pinto")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if s1 != "San Martín" || s2 != "Pinto" {
		t.Errorf("got %q y %q", s1, s2)
	}
}

func TestParseIntersection_ampersandSeparator(t *testing.T) {
	s1, s2, err := parseIntersection("9 de Julio & Constitución")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if s1 != "9 de Julio" || s2 != "Constitución" {
		t.Errorf("got %q y %q", s1, s2)
	}
}

func TestParseIntersection_dashSeparator(t *testing.T) {
	s1, s2, err := parseIntersection("Belgrano - Rivadavia")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if s1 != "Belgrano" || s2 != "Rivadavia" {
		t.Errorf("got %q y %q", s1, s2)
	}
}

func TestParseIntersection_trimSpaces(t *testing.T) {
	// Espacios extra alrededor de los nombres
	s1, s2, err := parseIntersection("  San Martín  /  Pinto  ")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if s1 != "San Martín" || s2 != "Pinto" {
		t.Errorf("espacios no recortados: got %q y %q", s1, s2)
	}
}

func TestParseIntersection_invalidFormat(t *testing.T) {
	_, _, err := parseIntersection("San Martín sin separador")
	if err == nil {
		t.Error("esperaba error para formato inválido, got nil")
	}
}

func TestParseIntersection_emptyString(t *testing.T) {
	_, _, err := parseIntersection("")
	if err == nil {
		t.Error("esperaba error para string vacío, got nil")
	}
}
