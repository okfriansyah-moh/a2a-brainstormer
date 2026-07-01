// Package markdown — publication-quality plan document generator (§8.31).
// Renders §5 Implementation Tasks in canonical task-block format suitable for
// AI-agent execution. Every section is deterministic: same state → same output.
package markdown

import (
	"encoding/json"
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GeneratePlan renders the consolidated implementation plan from s.
// The H1 format is always: # {title} — Implementation Plan
func GeneratePlan(s state.CanonicalState) (string, error) {
	var b strings.Builder
	title := shortTitle(s)

	// ── H1 + metadata block ──────────────────────────────────────────────────
	b.WriteString(fmt.Sprintf("# %s — Implementation Plan\n\n", title))
	b.WriteString(fmt.Sprintf("> %s\n", oneLineDescription(s)))
	b.WriteString(fmt.Sprintf("> Iteration %d · Confidence %.4f\n\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	// ── § 1. Goals ───────────────────────────────────────────────────────────
	b.WriteString("## 1. Goals\n\n")
	b.WriteString(renderPlanGoals(s))

	// ── § 2. Milestones ──────────────────────────────────────────────────────
	b.WriteString("## 2. Milestones\n\n")
	b.WriteString(renderMilestonesTable(s))

	// ── § 3. Assumptions ─────────────────────────────────────────────────────
	b.WriteString("## 3. Assumptions\n\n")
	b.WriteString(renderPlanAssumptions(s))

	// ── § 4. Dependency Graph ────────────────────────────────────────────────
	b.WriteString("## 4. Dependency Graph\n\n")
	b.WriteString(renderDependencyGraph(s))

	// ── § 5. Implementation Tasks ────────────────────────────────────────────
	b.WriteString("## 5. Implementation Tasks\n\n")
	b.WriteString(renderImplementationTasks(s))

	// ── § 6. Task Summary ────────────────────────────────────────────────────
	b.WriteString("## 6. Task Summary\n\n")
	b.WriteString(renderTaskSummaryTable(s))

	// ── § 6. Risks ───────────────────────────────────────────────────────────
	b.WriteString("### Risks\n\n")
	b.WriteString(renderDeduplicatedRisks(s))

	if len(s.OpenQuestions) > 0 {
		b.WriteString("### Open Questions\n\n")
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	}

	// ── § 7. How to Use This Plan ────────────────────────────────────────────
	b.WriteString("## 7. How to Use This Plan\n\n")
	b.WriteString(renderHowToUsePlan())

	// ── § 8. Deep Knowledge Reference ───────────────────────────────────────
	b.WriteString("## 8. Deep Knowledge Reference\n\n")
	b.WriteString(renderDeepKnowledgeReference(s))

	// ── For AI Agents appendix ───────────────────────────────────────────────
	b.WriteString(renderForAIAgentsAppendix(s, "plan"))

	return b.String(), nil
}

// ─── §1 Goals ─────────────────────────────────────────────────────────────────

// renderPlanGoals renders the §1 Goals section.
// Priority: s.Idea["goals"] → s.Idea["mvp_must_haves"] → fallback.
// NEVER emits "_Goals not yet defined by the agents._"
func renderPlanGoals(s state.CanonicalState) string {
	if v, ok := s.Idea["goals"]; ok {
		if goals := stringsFromAny(v); len(goals) > 0 {
			var b strings.Builder
			for _, g := range goals {
				b.WriteString(fmt.Sprintf("- %s\n", g))
			}
			b.WriteString("\n")
			return b.String()
		}
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" {
			return str + "\n\n"
		}
	}
	if mvp, ok := s.Idea["mvp_must_haves"]; ok {
		if items := stringsFromAny(mvp); len(items) > 0 {
			var b strings.Builder
			for _, item := range items {
				b.WriteString(fmt.Sprintf("- %s\n", item))
			}
			b.WriteString("\n")
			return b.String()
		}
	}
	return "_Goals not populated — run at least one iteration before finalizing_\n\n"
}

// ─── §2 Milestones ────────────────────────────────────────────────────────────

func renderMilestonesTable(s state.CanonicalState) string {
	if len(s.ExecutionPlan) == 0 {
		return "_No execution plan steps recorded yet._\n\n"
	}
	rows := make([][]string, 0, len(s.ExecutionPlan))
	for i, step := range s.ExecutionPlan {
		desc := step.Description
		if desc == "" {
			desc = step.Objective
		}
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			step.Title,
			desc,
		})
	}
	return renderTable([]string{"#", "Task", "Summary"}, rows)
}

// ─── §3 Assumptions ───────────────────────────────────────────────────────────

func renderPlanAssumptions(s state.CanonicalState) string {
	if len(s.Assumptions) == 0 {
		return "_No assumptions recorded yet._\n\n"
	}
	var b strings.Builder
	for _, a := range s.Assumptions {
		b.WriteString(fmt.Sprintf("- %s\n", a))
	}
	b.WriteString("\n")
	return b.String()
}

// ─── §4 Dependency Graph ──────────────────────────────────────────────────────

// renderDependencyGraph renders the ASCII dependency graph.
// Primary: s.Architecture["dependency_graph_ascii"]; fallback: linear chain.
func renderDependencyGraph(s state.CanonicalState) string {
	if v, ok := s.Architecture["dependency_graph_ascii"]; ok {
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str != "" && str != "<nil>" {
			return "```\n" + str + "\n```\n\n"
		}
	}
	// Linear chain fallback
	if len(s.ExecutionPlan) == 0 {
		return "_No dependency graph available yet._\n\n"
	}
	var b strings.Builder
	b.WriteString("```\n")
	for i, step := range s.ExecutionPlan {
		name := step.Title
		if len(name) > 30 {
			name = name[:30]
		}
		if i == 0 {
			b.WriteString(fmt.Sprintf("┌─ Task %d: %s\n", i+1, name))
		} else {
			b.WriteString(fmt.Sprintf("│\n▼\n┌─ Task %d: %s\n", i+1, name))
		}
		if i == len(s.ExecutionPlan)-1 {
			b.WriteString("└─ (done)\n")
		}
	}
	b.WriteString("```\n\n")
	return b.String()
}

// ─── §5 Implementation Tasks ──────────────────────────────────────────────────

// renderImplementationTasks renders each execution plan step as a canonical
// task block with all seven required fields.
func renderImplementationTasks(s state.CanonicalState) string {
	if len(s.ExecutionPlan) == 0 {
		return "_No implementation tasks defined yet._\n\n"
	}
	var b strings.Builder
	for i, step := range s.ExecutionPlan {
		b.WriteString(fmt.Sprintf("### Task %d — %s\n\n", i+1, step.Title))
		b.WriteString(renderTaskGoal(step))
		b.WriteString(renderLayersAffected(step))
		b.WriteString(renderFilesToCreate(step))
		b.WriteString(renderCodingStandards(s, i, step))
		b.WriteString(renderValidation(step))
		b.WriteString(renderInvariantCheck(s, i, step))
		b.WriteString(renderPromptContext(s, i))
		b.WriteString("---\n\n")
	}
	return b.String()
}

// renderTaskGoal returns the **Goal:** line for one step.
// Reads Objective first sentence ≤120 chars; falls back to Description first sentence.
func renderTaskGoal(step state.Step) string {
	text := ""
	if step.Objective != "" {
		text = firstSentence(step.Objective, 120)
	} else if step.Description != "" {
		text = firstSentence(step.Description, 120)
	}
	if text == "" {
		text = step.Title
	}
	return fmt.Sprintf("**Goal:** %s\n\n", text)
}

// renderLayersAffected derives unique module path prefixes from step.Deliverables.
func renderLayersAffected(step state.Step) string {
	if len(step.Deliverables) == 0 {
		return "**Layer(s) affected:** `backend/internal/modules/`\n\n"
	}
	seen := make(map[string]struct{})
	var paths []string
	for _, d := range step.Deliverables {
		prefix := extractLayerPrefix(d)
		if prefix != "" {
			if _, ok := seen[prefix]; !ok {
				seen[prefix] = struct{}{}
				paths = append(paths, prefix)
			}
		}
	}
	if len(paths) == 0 {
		return "**Layer(s) affected:** `backend/internal/modules/`\n\n"
	}
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = "`" + p + "`"
	}
	return fmt.Sprintf("**Layer(s) affected:** %s\n\n", strings.Join(quoted, ", "))
}

// extractLayerPrefix extracts the relevant layer prefix from a file path.
// Returns the first segment that matches known layer patterns.
func extractLayerPrefix(path string) string {
	for _, prefix := range []string{
		"backend/internal/modules/",
		"backend/internal/platform/",
		"backend/cmd/",
		"agent/internal/",
		"agent/cmd/",
		"frontend/src/",
	} {
		if strings.HasPrefix(path, prefix) {
			// Return up to the next path component after the prefix.
			rest := path[len(prefix):]
			idx := strings.Index(rest, "/")
			if idx >= 0 {
				return prefix + rest[:idx] + "/"
			}
			return prefix
		}
	}
	// Fall back to the first two path components.
	parts := strings.SplitN(path, "/", 3)
	if len(parts) >= 2 {
		return strings.Join(parts[:2], "/") + "/"
	}
	return path
}

// renderFilesToCreate converts step.Deliverables into a bullet list with
// function contracts as sub-bullets and failure handling at the end.
func renderFilesToCreate(step state.Step) string {
	var b strings.Builder
	b.WriteString("**Files to create:**\n")
	if len(step.Deliverables) == 0 {
		b.WriteString("- _No explicit deliverables listed_\n")
	} else {
		for _, d := range step.Deliverables {
			b.WriteString(fmt.Sprintf("- `%s`\n", d))
		}
		for _, c := range step.FunctionContracts {
			b.WriteString(fmt.Sprintf("  - `%s`\n", c))
		}
		if step.FailureHandling != "" {
			b.WriteString(fmt.Sprintf("  - **Failure handling:** %s\n", step.FailureHandling))
		}
	}
	b.WriteString("\n")
	return b.String()
}

// renderCodingStandards renders coding standards from the phase enrichment overlay
// or falls back to canonical generic standards.
func renderCodingStandards(s state.CanonicalState, idx int, step state.Step) string {
	var b strings.Builder
	b.WriteString("**Coding standards:**\n")
	enrichment := phaseEnrichmentAt(s, idx)
	for _, cs := range enrichment.CodingStandards {
		b.WriteString(fmt.Sprintf("- %s\n", cs))
	}
	if stepUsesGo(step) {
		b.WriteString("- SRP: Each file has a single, well-named responsibility\n")
		b.WriteString("- DIP: Depend on interfaces, not concrete types\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderValidation renders the **Validation:** section.
// Always includes go build and go vet. Appends frontend check when frontend
// deliverables are present. Adds ExitCriteria as [ ] checkboxes.
// NEVER emits "per module test suite".
func renderValidation(step state.Step) string {
	var b strings.Builder
	b.WriteString("**Validation:**\n")
	if stepUsesGo(step) {
		b.WriteString("- `cd backend && go build ./...`: zero build errors\n")
		b.WriteString("- `cd backend && go vet ./...`: zero vet issues\n")
	}

	// Add frontend check when a frontend deliverable is present.
	hasFrontend := false
	for _, d := range step.Deliverables {
		if strings.HasPrefix(d, "frontend/") {
			hasFrontend = true
			break
		}
	}
	if hasFrontend {
		b.WriteString("- `cd frontend && pnpm check && pnpm build`: zero errors\n")
	}

	// Render ExitCriteria as behavioural assertions (not commands).
	for _, c := range step.ExitCriteria {
		b.WriteString(fmt.Sprintf("- [ ] %s\n", c))
	}
	b.WriteString("\n")
	return b.String()
}

// renderInvariantCheck renders the **Invariant check:** section with enricher
// items followed by the 4 canonical invariants (unconditionally).
func renderInvariantCheck(s state.CanonicalState, idx int, step state.Step) string {
	var b strings.Builder
	b.WriteString("**Invariant check:**\n")
	enrichment := phaseEnrichmentAt(s, idx)
	for _, ic := range enrichment.InvariantChecks {
		b.WriteString(fmt.Sprintf("- [ ] %s\n", ic))
	}
	b.WriteString("- [ ] No file modified outside \"Files to create/modify\" list\n")
	if stepUsesGo(step) {
		b.WriteString("- [ ] No cross-module import introduced\n")
		b.WriteString("- [ ] No `os.Getenv` outside `config.go`\n")
		b.WriteString("- [ ] All SQL via parameterized queries only (if DB-touching)\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderPromptContext renders the **Prompt context needed:** line.
// Reads from phase enrichment overlay; falls back to canonical refs.
func renderPromptContext(s state.CanonicalState, idx int) string {
	enrichment := phaseEnrichmentAt(s, idx)
	if len(enrichment.PromptContextRefs) > 0 {
		return fmt.Sprintf("**Prompt context needed:** %s\n\n",
			strings.Join(enrichment.PromptContextRefs, ", "))
	}
	return "**Prompt context needed:** Product architecture, execution plan step objectives, and clarifying questions from the brainstorm session\n\n"
}

// ─── §6 Task Summary ──────────────────────────────────────────────────────────

func renderTaskSummaryTable(s state.CanonicalState) string {
	if len(s.ExecutionPlan) == 0 {
		return "_No tasks defined yet._\n\n"
	}
	rows := make([][]string, 0, len(s.ExecutionPlan))
	for i, step := range s.ExecutionPlan {
		primaryFile := ""
		if len(step.Deliverables) > 0 {
			primaryFile = step.Deliverables[0]
		}
		deps := "_none_"
		if len(step.BlockingDependencies) > 0 {
			deps = strings.Join(step.BlockingDependencies, ", ")
		}
		notes := ""
		if step.Scope != "" {
			notes = truncateChars(step.Scope, 60)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			step.Title,
			primaryFile,
			deps,
			notes,
		})
	}
	return renderTable([]string{"#", "Task Name", "Primary File", "Depends-On", "Notes"}, rows)
}

// renderDeduplicatedRisks renders the risks table.
func renderDeduplicatedRisks(s state.CanonicalState) string {
	return renderRisksTable(s)
}

// ─── §7 How to Use This Plan ──────────────────────────────────────────────────

func renderHowToUsePlan() string {
	return `**Task Execution Protocol**

A task is NOT complete until:

1. All **Validation** commands produce zero output,
2. All **Invariant check** boxes are ticked,
3. No file outside **Files to create/modify** was touched.

**Quality Gate Sequence**

Run the validation commands listed in each task block in order. Each must return zero findings before proceeding to the next task.

**Execution Rules**

- Complete tasks in dependency order (see §4 Dependency Graph).
- Each task maps to one focused implementation session.
- Update risks and open questions when new information emerges during build-out.

`
}

// ─── §8 Deep Knowledge Reference ─────────────────────────────────────────────

func renderDeepKnowledgeReference(s state.CanonicalState) string {
	var b strings.Builder

	b.WriteString("Domain-specific schemas, algorithms, and contracts captured during the brainstorm session.\n\n")

	// Tech stack rationale from Architecture.
	if v, ok := s.Architecture["tech_stack_rationale"]; ok {
		if m, ok := v.(map[string]any); ok && len(m) > 0 {
			b.WriteString("### Tech Stack Rationale\n\n")
			writeMap(&b, m)
		}
	}

	// Module responsibility table.
	b.WriteString("### Module Responsibilities\n\n")
	b.WriteString(renderModuleResponsibilityTable(s))

	// Deep knowledge sections from enricher.
	if v, ok := s.Architecture["deep_knowledge_sections"]; ok {
		sections := extractDKSections(v)
		if len(sections) > 0 {
			for _, dks := range sections {
				b.WriteString(fmt.Sprintf("### %s\n\n", dks.heading))
				b.WriteString(dks.content + "\n\n")
			}
		}
	}

	return b.String()
}

type dkSection struct {
	heading string
	content string
}

func extractDKSections(v any) []dkSection {
	maps := mapsFromAny(v)
	if len(maps) == 0 {
		return nil
	}
	out := make([]dkSection, 0, len(maps))
	for _, m := range maps {
		heading := strings.TrimSpace(fmt.Sprintf("%v", m["heading"]))
		content := strings.TrimSpace(fmt.Sprintf("%v", m["content"]))
		if heading == "" || heading == "<nil>" {
			continue
		}
		if content == "<nil>" {
			content = ""
		}
		out = append(out, dkSection{heading: heading, content: content})
	}
	return out
}

func renderModuleResponsibilityTable(s state.CanonicalState) string {
	rows := [][]string{}

	// Override or extend from actual architecture if present.
	if v, ok := s.Architecture["layers"]; ok {
		if layers := mapsFromAny(v); len(layers) > 0 {
			rows = make([][]string, 0, len(layers))
			for _, layer := range layers {
				name := strings.TrimSpace(fmt.Sprintf("%v", layer["name"]))
				resp := strings.TrimSpace(fmt.Sprintf("%v", layer["responsibility"]))
				if name == "" || name == "<nil>" {
					continue
				}
				if resp == "<nil>" {
					resp = "—"
				}
				rows = append(rows, []string{name, resp})
			}
		}
	}
	if len(rows) == 0 {
		return "_Module responsibilities will appear here once architecture.layers is defined in the brainstorm session._\n\n"
	}
	return renderTable([]string{"Module Path", "Responsibility"}, rows)
}

// ─── Phase enrichment helpers ─────────────────────────────────────────────────

// phaseEnrichmentAt reads s.Architecture["plan_phase_enrichments"] and returns
// the PhaseEnrichment at the given 0-based index, or a zero value when absent.
// The data is stored as []any of map[string]any after JSON round-trip.
func phaseEnrichmentAt(s state.CanonicalState, idx int) phaseEnrichmentData {
	v, ok := s.Architecture["plan_phase_enrichments"]
	if !ok {
		return phaseEnrichmentData{}
	}
	items := extractPhaseEnrichments(v)
	if idx < 0 || idx >= len(items) {
		return phaseEnrichmentData{}
	}
	return items[idx]
}

type phaseEnrichmentData struct {
	CodingStandards   []string
	InvariantChecks   []string
	LayerTags         []string
	PromptContextRefs []string
}

func extractPhaseEnrichments(v any) []phaseEnrichmentData {
	// The data arrives as []any (JSON-decoded maps) after the arch map round-trip.
	// Re-encode to JSON and decode into typed structs.
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	// Try as typed array first.
	type wire struct {
		Position          int      `json:"position"`
		CodingStandards   []string `json:"coding_standards"`
		InvariantChecks   []string `json:"invariant_checks"`
		LayerTags         []string `json:"layer_tags"`
		PromptContextRefs []string `json:"prompt_context_refs"`
	}
	var wires []wire
	if err := json.Unmarshal(raw, &wires); err != nil {
		return nil
	}
	// Sort by position to ensure the idx lookup is correct.
	out := make([]phaseEnrichmentData, len(wires))
	for _, w := range wires {
		pos := w.Position
		if pos < 0 || pos >= len(wires) {
			continue
		}
		out[pos] = phaseEnrichmentData{
			CodingStandards:   w.CodingStandards,
			InvariantChecks:   w.InvariantChecks,
			LayerTags:         w.LayerTags,
			PromptContextRefs: w.PromptContextRefs,
		}
	}
	return out
}

// stepUsesGo reports whether a task step targets Go/backend deliverables.
func stepUsesGo(step state.Step) bool {
	for _, d := range step.Deliverables {
		if strings.HasSuffix(d, ".go") || strings.HasPrefix(d, "backend/") || strings.HasPrefix(d, "agent/") {
			return true
		}
	}
	return false
}
