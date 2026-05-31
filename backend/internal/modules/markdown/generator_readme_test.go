// Package markdown — tests for the GenerateReadme long-form generator.
package markdown

import (
	"strings"
	"testing"
)

func TestGenerateReadme_NonEmpty(t *testing.T) {
	s := sampleState()
	got, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("GenerateReadme returned error: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(got, " — README") {
		t.Errorf("expected '— README' in title")
	}
}

func TestGenerateReadme_Determinism(t *testing.T) {
	s := sampleState()
	got1, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	got2, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got1 != got2 {
		t.Error("GenerateReadme is not deterministic: two calls produced different output")
	}
}

func TestGenerateReadme_StructuralSections(t *testing.T) {
	s := sampleState()
	got, err := GenerateReadme(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"## Overview",
		"## Quick Start",
		"## Configuration",
		"## Roadmap",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}
