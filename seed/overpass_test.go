// seed/overpass_test.go
package seed

import (
	"testing"
)

func TestBuildName_twoStreets(t *testing.T) {
	nodeStreets := map[int64][]string{
		1: {"San Martín", "Pinto"},
	}
	name := buildName(1, nodeStreets)
	if name != "San Martín & Pinto" {
		t.Errorf("got %q, want %q", name, "San Martín & Pinto")
	}
}

func TestBuildName_oneStreet(t *testing.T) {
	nodeStreets := map[int64][]string{
		1: {"San Martín"},
	}
	name := buildName(1, nodeStreets)
	if name != "San Martín" {
		t.Errorf("got %q, want %q", name, "San Martín")
	}
}

func TestBuildName_noStreets(t *testing.T) {
	nodeStreets := map[int64][]string{}
	name := buildName(42, nodeStreets)
	expected := "nodo 42"
	if name != expected {
		t.Errorf("got %q, want %q", name, expected)
	}
}

func TestIntersectionType_twoStreets(t *testing.T) {
	nodeStreets := map[int64][]string{
		1: {"San Martín", "Pinto"},
	}
	typ := intersectionType(1, nodeStreets)
	if typ != "intersection" {
		t.Errorf("got %q, want %q", typ, "intersection")
	}
}

func TestIntersectionType_oneStreet(t *testing.T) {
	nodeStreets := map[int64][]string{
		1: {"San Martín"},
	}
	typ := intersectionType(1, nodeStreets)
	if typ != "dead_end" {
		t.Errorf("got %q, want %q", typ, "dead_end")
	}
}

func TestParseSpeed_kmh(t *testing.T) {
	if s := parseSpeed("50"); s != 50 {
		t.Errorf("got %.1f, want 50.0", s)
	}
}

func TestParseSpeed_mph(t *testing.T) {
	s := parseSpeed("30 mph")
	expected := 30 * 1.60934
	if s < expected-0.1 || s > expected+0.1 {
		t.Errorf("got %.2f, want %.2f", s, expected)
	}
}

func TestParseSpeed_walk(t *testing.T) {
	if s := parseSpeed("walk"); s != 20 {
		t.Errorf("got %.1f, want 20.0", s)
	}
}

func TestParseSpeed_empty(t *testing.T) {
	if s := parseSpeed(""); s != 20 {
		t.Errorf("got %.1f, want 20.0", s)
	}
}

func TestParseSpeed_invalid(t *testing.T) {
	if s := parseSpeed("abc"); s != 40 {
		t.Errorf("got %.1f, want 40.0", s)
	}
}
