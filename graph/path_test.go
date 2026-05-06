// graph/path_test.go
package graph

import (
	"testing"
)

// --- CompressSteps ---

func TestCompressSteps_empty(t *testing.T) {
	result := CompressSteps([]RouteStep{})
	if len(result) != 0 {
		t.Errorf("esperaba 0 steps, got %d", len(result))
	}
}

func TestCompressSteps_singleStep(t *testing.T) {
	steps := []RouteStep{
		{From: "A", To: "B", Street: "San Martín"},
	}
	result := CompressSteps(steps)
	if len(result) != 1 {
		t.Fatalf("esperaba 1 step, got %d", len(result))
	}
	if result[0].From != "A" || result[0].To != "B" {
		t.Errorf("step incorrecto: %+v", result[0])
	}
}

func TestCompressSteps_consecutiveSameStreet(t *testing.T) {
	steps := []RouteStep{
		{From: "A", To: "B", Street: "San Martín"},
		{From: "B", To: "C", Street: "San Martín"},
		{From: "C", To: "D", Street: "San Martín"},
	}
	result := CompressSteps(steps)

	if len(result) != 1 {
		t.Fatalf("esperaba 1 step comprimido, got %d", len(result))
	}
	if result[0].From != "A" {
		t.Errorf("From incorrecto: got %q, want %q", result[0].From, "A")
	}
	if result[0].To != "D" {
		t.Errorf("To incorrecto: got %q, want %q", result[0].To, "D")
	}
	if result[0].Street != "San Martín" {
		t.Errorf("Street incorrecto: got %q", result[0].Street)
	}
}

func TestCompressSteps_differentStreets(t *testing.T) {
	steps := []RouteStep{
		{From: "A", To: "B", Street: "San Martín"},
		{From: "B", To: "C", Street: "Pinto"},
		{From: "C", To: "D", Street: "9 de Julio"},
	}
	result := CompressSteps(steps)

	if len(result) != 3 {
		t.Fatalf("esperaba 3 steps, got %d", len(result))
	}
}

func TestCompressSteps_mixedStreets(t *testing.T) {
	// Dos tramos por San Martín, uno por Pinto, dos por 9 de Julio
	steps := []RouteStep{
		{From: "A", To: "B", Street: "San Martín"},
		{From: "B", To: "C", Street: "San Martín"},
		{From: "C", To: "D", Street: "Pinto"},
		{From: "D", To: "E", Street: "9 de Julio"},
		{From: "E", To: "F", Street: "9 de Julio"},
	}
	result := CompressSteps(steps)

	if len(result) != 3 {
		t.Fatalf("esperaba 3 steps, got %d", len(result))
	}
	if result[0].From != "A" || result[0].To != "C" || result[0].Street != "San Martín" {
		t.Errorf("primer step incorrecto: %+v", result[0])
	}
	if result[1].From != "C" || result[1].To != "D" || result[1].Street != "Pinto" {
		t.Errorf("segundo step incorrecto: %+v", result[1])
	}
	if result[2].From != "D" || result[2].To != "F" || result[2].Street != "9 de Julio" {
		t.Errorf("tercer step incorrecto: %+v", result[2])
	}
}
