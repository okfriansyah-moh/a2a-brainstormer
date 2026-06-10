package session

import (
	"testing"
)

func TestValidateOutputDocs_RejectsRoadmap(t *testing.T) {
	err := validateOutputDocs([]string{"architecture", "roadmap"})
	if err == nil {
		t.Fatal("expected roadmap key to be rejected")
	}
}

func TestDefaultOutputDocs_PlanOnly(t *testing.T) {
	if len(DefaultOutputDocs) != 2 {
		t.Fatalf("DefaultOutputDocs = %v", DefaultOutputDocs)
	}
	if DefaultOutputDocs[0] != "architecture" || DefaultOutputDocs[1] != "plan" {
		t.Fatalf("unexpected defaults: %v", DefaultOutputDocs)
	}
	if AllowedOutputDocs["roadmap"] {
		t.Fatal("roadmap should not be allowed")
	}
}

func TestMapArtifactSource(t *testing.T) {
	cases := map[string]string{
		"ai":          "ai",
		"ai_fallback": "hybrid",
		"hybrid":      "hybrid",
		"deterministic": "deterministic",
		"":            "deterministic",
	}
	for in, want := range cases {
		if got := mapArtifactSource(in); got != want {
			t.Fatalf("mapArtifactSource(%q) = %q, want %q", in, got, want)
		}
	}
}
