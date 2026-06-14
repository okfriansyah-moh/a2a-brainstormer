package state_test

import (
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

func TestEnrichIdeaFromSession_BackfillsTextAndTitle(t *testing.T) {
	cs := state.CanonicalState{
		Idea: map[string]any{"summary": "agent-only delta"},
	}
	sessionIdea := "i have an idea where i want to build an agentic commerce platform"

	out := state.EnrichIdeaFromSession(cs, sessionIdea)

	if got := out.Idea["text"]; got != sessionIdea {
		t.Fatalf("text = %v, want session idea", got)
	}
	title, ok := out.Idea["title"].(string)
	if !ok || title == "" {
		t.Fatalf("expected derived title, got %v", out.Idea["title"])
	}
	if title == "Untitled Brainstorm" {
		t.Fatalf("title should be summarized from idea, got %q", title)
	}
}

func TestSummarizeIdeaTitle_StripsPreamble(t *testing.T) {
	got := state.SummarizeIdeaTitle("i want to build an agentic commerce rental marketplace")
	if got == "" {
		t.Fatal("expected non-empty title")
	}
	if got == "i want to build an agentic commerce rental marketplace" {
		t.Fatalf("expected preamble stripped, got %q", got)
	}
}

func TestMerge_PreservesBaseIdeaTextWhenIncomingSparse(t *testing.T) {
	base := state.CanonicalState{
		Idea: map[string]any{
			"text": "i have an idea where i want to build agentic commerce",
		},
	}
	incoming := state.CanonicalState{
		Idea: map[string]any{
			"summary": "Refined marketplace concept",
		},
	}
	out := state.Merge(base, incoming)
	if out.Idea["text"] != base.Idea["text"] {
		t.Fatalf("text lost after merge: %v", out.Idea["text"])
	}
	if out.Idea["summary"] != incoming.Idea["summary"] {
		t.Fatalf("summary not merged: %v", out.Idea["summary"])
	}
}
