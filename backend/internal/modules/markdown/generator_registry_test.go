// Package markdown — tests for the GenerateAll registry function.
package markdown

import (
	"strings"
	"testing"
)

func TestGenerateAll_AllFourKeys(t *testing.T) {
	s := sampleState()
	keys := []string{"architecture", "roadmap", "plan", "readme"}
	result, err := GenerateAll(s, keys)
	if err != nil {
		t.Fatalf("GenerateAll returned error: %v", err)
	}
	for _, key := range keys {
		doc, ok := result[key]
		if !ok {
			t.Errorf("result missing key %q", key)
			continue
		}
		if doc.Content == "" {
			t.Errorf("key %q: empty content", key)
		}
		if doc.LineCount < 1000 {
			t.Errorf("key %q: expected ≥ 1000 lines, got %d", key, doc.LineCount)
		}
		if doc.Filename == "" {
			t.Errorf("key %q: empty filename", key)
		}
	}
}

func TestGenerateAll_UnknownKeyError(t *testing.T) {
	s := sampleState()
	_, err := GenerateAll(s, []string{"architecture", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error message should mention the unknown key, got: %v", err)
	}
}

func TestGenerateAll_OrderPreserved(t *testing.T) {
	s := sampleState()
	keys := []string{"readme", "plan"}
	result, err := GenerateAll(s, keys)
	if err != nil {
		t.Fatalf("GenerateAll returned error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries in result, got %d", len(result))
	}
	for _, key := range keys {
		if _, ok := result[key]; !ok {
			t.Errorf("result missing key %q", key)
		}
	}
}

func TestGenerateAll_LineCountMatchesContent(t *testing.T) {
	s := sampleState()
	result, err := GenerateAll(s, []string{"architecture"})
	if err != nil {
		t.Fatalf("GenerateAll returned error: %v", err)
	}
	doc := result["architecture"]
	expectedLines := strings.Count(doc.Content, "\n") + 1
	if doc.LineCount != expectedLines {
		t.Errorf("LineCount mismatch: stored %d, counted %d", doc.LineCount, expectedLines)
	}
}
