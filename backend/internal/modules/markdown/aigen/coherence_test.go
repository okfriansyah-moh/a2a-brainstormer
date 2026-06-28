package aigen_test

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/markdown/aigen"
)

func TestApplyCoherenceGuardrails_RevertsShrunkSection(t *testing.T) {
	before := "## 1. Goals\n\n" + strings.Repeat("long content ", 50) + "\n"
	after := "## 1. Goals\n\nshort\n"
	rubric := aigen.Rubric{Sections: []aigen.SectionRule{{Heading: "1. Goals", MinChars: 10}}}
	got, err := aigen.ApplyCoherenceGuardrails(before, after, rubric, nil)
	if err != nil {
		t.Fatalf("ApplyCoherenceGuardrails: %v", err)
	}
	body, ok := aigen.ExtractSectionBody(got, "1. Goals")
	if !ok || len(strings.TrimSpace(body)) < 100 {
		t.Fatalf("expected reverted long body, got %q", body)
	}
}

func TestApplyCoherenceGuardrails_StructuralFreeze(t *testing.T) {
	before := "## 1. Goals\n\na\n"
	after := "## 1. Renamed\n\nb\n"
	rubric := aigen.Rubric{Sections: []aigen.SectionRule{{Heading: "1. Goals", MinChars: 1}}}
	_, err := aigen.ApplyCoherenceGuardrails(before, after, rubric, nil)
	if err == nil {
		t.Fatal("expected structural freeze error")
	}
}
