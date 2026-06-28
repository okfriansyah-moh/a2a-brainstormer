package aigen

import "testing"

func TestParseCoherenceAudit_EmptyFindings(t *testing.T) {
	raw := `{"findings":[]}`
	findings, err := parseCoherenceAudit(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}
