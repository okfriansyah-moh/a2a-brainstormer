// Package aigen — domain-focus helpers keep generated documents anchored to
// the user's product idea from the brainstorm session, not the A2A Brainstorm
// tool that produced the session.
package aigen

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// domainFocusRules returns system-prompt instructions that prevent the LLM from
// rewriting user documents as specs for the brainstorm orchestrator.
func domainFocusRules(s state.CanonicalState) string {
	title := productTitle(s)
	oneLiner := productOneLiner(s)
	var sb strings.Builder
	sb.WriteString("## Product focus (CRITICAL — overrides generic engineering patterns)\n\n")
	sb.WriteString("You are documenting the USER'S PRODUCT from the brainstorm session — NOT the A2A Brainstorm tool, agent pipeline, or session orchestrator that generated this file.\n\n")
	if title != "" {
		sb.WriteString("Product name: ")
		sb.WriteString(title)
		sb.WriteString("\n")
	}
	if oneLiner != "" {
		sb.WriteString("Product summary: ")
		sb.WriteString(oneLiner)
		sb.WriteString("\n\n")
	}
	sb.WriteString(`Rules:
1. Every section must address the user's domain problem, users, solution, scope, risks, assumptions, open questions, and execution_plan from the canonical state.
2. Preserve every domain-specific fact from the deterministic scaffold. Expand with detail drawn from canonical state — never replace domain content with generic platform boilerplate.
3. Task blocks must reflect the execution_plan steps (titles, deliverables, objectives) — not a generic multi-agent framework roadmap.
4. Tech stack, APIs, data models, and module names must match architecture and execution_plan in canonical state.

FORBIDDEN unless explicitly part of the user's product in canonical state:
- A2A protocol, a2a-go, Agent-to-Agent messaging, DataPart artifacts
- Brainstorm sessions, iteration engine, convergence engine, canonical state merge
- This repository's modules (session, iteration, agent, state, convergence, markdown generators)
- "deterministic multi-agent design system" as the subject of the document
- Generic Go monolith scaffolding (LLMProvider, vertical-slice under backend/internal/modules/) when not in execution_plan deliverables

The "## For AI Agents" appendix (when present) must describe implementing THIS product — stack, contracts, and build order from canonical state — not the brainstorm tool internals.
`)
	return sb.String()
}

func productTitle(s state.CanonicalState) string {
	for _, key := range []string{"title", "name"} {
		if v := stringFromMap(s.Idea, key); v != "" {
			return v
		}
	}
	if v := stringFromMap(s.Architecture, "title"); v != "" {
		return v
	}
	if v := stringFromMap(s.Idea, "text"); v != "" {
		return firstLine(v, 80)
	}
	return ""
}

func productOneLiner(s state.CanonicalState) string {
	for _, key := range []string{"value_proposition", "summary", "solution_summary", "problem_statement"} {
		if v := stringFromMap(s.Idea, key); v != "" {
			return truncateRunes(v, 300)
		}
	}
	for _, key := range []string{"solution_summary", "problem_statement"} {
		if v := stringFromMap(s.Architecture, key); v != "" {
			return truncateRunes(v, 300)
		}
	}
	if v := stringFromMap(s.Idea, "text"); v != "" {
		return truncateRunes(v, 300)
	}
	return ""
}

func firstLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexAny(s, "\n\r"); idx >= 0 {
		s = s[:idx]
	}
	return truncateRunes(s, max)
}

func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max]) + "…"
}

// summariseState renders canonical-state context for AI prompts. Richer than a
// title-only summary so section expansion stays anchored to the user's domain.
func summariseState(s state.CanonicalState) string {
	var sb strings.Builder
	if title := productTitle(s); title != "" {
		sb.WriteString("Product: ")
		sb.WriteString(title)
		sb.WriteString("\n")
	}
	if one := productOneLiner(s); one != "" {
		sb.WriteString("Summary: ")
		sb.WriteString(one)
		sb.WriteString("\n")
	}
	for _, field := range []string{"problem", "target_users", "value_proposition", "mvp_must_haves"} {
		if v := stringFromMap(s.Idea, field); v != "" {
			sb.WriteString(capitaliseLabel(field))
			sb.WriteString(": ")
			sb.WriteString(v)
			sb.WriteString("\n")
		}
	}
	if v := stringFromMap(s.Architecture, "problem_statement"); v != "" {
		sb.WriteString("Architecture problem: ")
		sb.WriteString(v)
		sb.WriteString("\n")
	}
	if v := stringFromMap(s.Architecture, "solution_summary"); v != "" {
		sb.WriteString("Architecture solution: ")
		sb.WriteString(v)
		sb.WriteString("\n")
	}
	if names := componentNames(s.Architecture); len(names) > 0 {
		sb.WriteString("Components: ")
		sb.WriteString(strings.Join(names, ", "))
		sb.WriteString("\n")
	}
	if len(s.ExecutionPlan) > 0 {
		sb.WriteString("Execution plan:\n")
		for i, step := range s.ExecutionPlan {
			line := step.Title
			if step.Objective != "" {
				line += " — " + truncateRunes(step.Objective, 120)
			} else if step.Description != "" {
				line += " — " + truncateRunes(step.Description, 120)
			}
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, line))
		}
	}
	if len(s.Assumptions) > 0 {
		sb.WriteString(fmt.Sprintf("Assumptions (%d): ", len(s.Assumptions)))
		limit := len(s.Assumptions)
		if limit > 6 {
			limit = 6
		}
		sb.WriteString(strings.Join(s.Assumptions[:limit], "; "))
		if len(s.Assumptions) > limit {
			sb.WriteString("; …")
		}
		sb.WriteString("\n")
	}
	if len(s.OpenQuestions) > 0 {
		sb.WriteString(fmt.Sprintf("Open questions (%d): ", len(s.OpenQuestions)))
		limit := len(s.OpenQuestions)
		if limit > 6 {
			limit = 6
		}
		sb.WriteString(strings.Join(s.OpenQuestions[:limit], "; "))
		if len(s.OpenQuestions) > limit {
			sb.WriteString("; …")
		}
		sb.WriteString("\n")
	}
	if len(s.Risks) > 0 {
		sb.WriteString(fmt.Sprintf("Risks: %d identified\n", len(s.Risks)))
	}
	return sb.String()
}
