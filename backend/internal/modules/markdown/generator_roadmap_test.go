// Package markdown — roadmap generator deprecated in v1.8.
package markdown

import (
	"errors"
	"testing"
)

func TestGenerateRoadmap_Deprecated(t *testing.T) {
	_, err := GenerateRoadmap(sampleState())
	if !errors.Is(err, ErrOutputKeyDeprecated) {
		t.Fatalf("expected ErrOutputKeyDeprecated, got %v", err)
	}
}
