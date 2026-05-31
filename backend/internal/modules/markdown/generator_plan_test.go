// Package markdown — tests for the GeneratePlan long-form generator.
package markdown

import (
	"strings"
	"testing"
)

func TestGeneratePlan_NonEmpty(t *testing.T) {
	s := sampleState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("GeneratePlan returned error: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(got, " — Implementation Plan") {
		t.Errorf("expected '— Implementation Plan' in title")
	}
}

func TestGeneratePlan_Determinism(t *testing.T) {
	s := sampleState()
	got1, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	got2, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got1 != got2 {
		t.Error("GeneratePlan is not deterministic: two calls produced different output")
	}
}

func TestGeneratePlan_StructuralSections(t *testing.T) {
	s := sampleState()
	got, err := GeneratePlan(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"## 1. Goal",
		"## 2. Architecture Overview",
		"## 3. Tech Stack",
		"## 4. Project Structure",
		"## 5. Implementation Tasks",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}
