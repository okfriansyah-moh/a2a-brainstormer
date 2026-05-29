// Package aigen — AI generator that rewrites deterministic scaffolds.
//
// Per docs/PLAN.md §8.27, the Generator takes the deterministic markdown
// scaffold as the "seed" payload, asks the configured LLMProvider to rewrite
// it into a richer document, then validates the result against a rubric.
// When validation fails, up to MaxRepairs follow-up "fix these findings"
// prompts are issued. If the final draft still fails, the per-document path
// falls back to the deterministic scaffold and emits an `aigen_fallback` warn
// log — no error is propagated in hybrid mode.
package aigen

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/llm"
	"a2a-brainstorm/backend/internal/shared"
)

// Mode controls how generation failures are reported to the caller.
type Mode int

const (
	// ModeHybrid silently falls back to the deterministic scaffold on any AI
	// failure (logged at warn level via `aigen_fallback`).
	ModeHybrid Mode = iota
	// ModeAI returns a wrapped error on any unrecoverable AI failure. Use only
	// when the operator explicitly opts in to strict AI-only generation.
	ModeAI
)

// Generator wraps an LLMProvider with skill-bundle composition and rubric
// auto-repair. It is safe for concurrent use across requests provided the
// underlying LLMProvider is.
type Generator struct {
	llm         llm.LLMProvider
	bundle      SkillBundle
	maxRepairs  int
	temperature float64
	mode        Mode
	logger      *slog.Logger
}

// New constructs a Generator. maxRepairs is clamped to [0,5]; temperature is
// clamped to [0.0,1.0]. A nil logger is replaced with slog.Default().
func New(provider llm.LLMProvider, bundle SkillBundle, maxRepairs int, temperature float64, mode Mode, logger *slog.Logger) *Generator {
	if maxRepairs < 0 {
		maxRepairs = 0
	}
	if maxRepairs > 5 {
		maxRepairs = 5
	}
	if temperature < 0 {
		temperature = 0
	}
	if temperature > 1 {
		temperature = 1
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Generator{
		llm:         provider,
		bundle:      bundle,
		maxRepairs:  maxRepairs,
		temperature: temperature,
		mode:        mode,
		logger:      logger,
	}
}

// Enhance walks scaffolds and attempts an AI rewrite for each in parallel
// (one goroutine per doc). Keys without an AI improvement are returned as the
// original scaffold (hybrid mode) or omitted with an error (ModeAI).
//
// Parallel dispatch is critical for the finalize endpoint: a single Claude
// Sonnet 4.6 turn through OpenCode can take ~5 min, so 4 sequential docs
// would exceed the operator's finalize timeout. Each LLM provider used here
// must be safe for concurrent calls (OpenCodeProvider and CopilotProvider are).
//
// The returned map always covers every key in scaffolds.
func (g *Generator) Enhance(ctx context.Context, s state.CanonicalState, scaffolds map[string]shared.GeneratedDocument) (map[string]shared.GeneratedDocument, error) {
	if g == nil || g.llm == nil {
		return scaffolds, nil
	}
	if len(g.bundle.Skills) == 0 {
		g.logger.WarnContext(ctx, "aigen_fallback",
			slog.String("reason", "empty skill bundle"),
		)
		if g.mode == ModeAI {
			return nil, errors.New("aigen: skill bundle is empty in ai mode")
		}
		return scaffolds, nil
	}

	keys := sortedKeys(scaffolds)

	type result struct {
		doc shared.GeneratedDocument
		err error
	}
	results := make(map[string]result, len(keys))
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, key := range keys {
		wg.Add(1)
		go func(k string, scaffold shared.GeneratedDocument) {
			defer wg.Done()
			enhanced, err := g.enhanceOne(ctx, k, scaffold, s)
			mu.Lock()
			results[k] = result{doc: enhanced, err: err}
			mu.Unlock()
		}(key, scaffolds[key])
	}
	wg.Wait()

	out := make(map[string]shared.GeneratedDocument, len(keys))
	for _, key := range keys {
		r := results[key]
		if r.err != nil {
			if g.mode == ModeAI {
				return nil, fmt.Errorf("aigen: %s: %w", key, r.err)
			}
			g.logger.WarnContext(ctx, "aigen_fallback",
				slog.String("doc_key", key),
				slog.String("reason", r.err.Error()),
			)
			fallback := scaffolds[key]
			fallback.Source = "ai_fallback"
			out[key] = fallback
			continue
		}
		out[key] = r.doc
	}
	return out, nil
}

// enhanceOne produces one AI-rewritten document. Returns the original scaffold
// wrapped as a synthetic error when the AI output cannot satisfy the rubric;
// the caller decides whether to surface the error or fall back.
func (g *Generator) enhanceOne(ctx context.Context, key string, scaffold shared.GeneratedDocument, s state.CanonicalState) (shared.GeneratedDocument, error) {
	rubric := RubricFor(key)
	systemPrompt := g.buildSystemPrompt(key)
	userPrompt := buildInitialUserPrompt(key, scaffold, s)

	resp, err := g.llm.Generate(ctx, llm.LLMRequest{
		SystemPrompt:   systemPrompt,
		UserMessage:    userPrompt,
		Temperature:    g.temperature,
		ResponseFormat: "text",
	})
	if err != nil {
		return shared.GeneratedDocument{}, fmt.Errorf("initial generate: %w", err)
	}
	draft := strings.TrimSpace(resp.Content)
	if draft == "" {
		return shared.GeneratedDocument{}, errors.New("initial draft was empty")
	}
	if len(draft) < len(strings.TrimSpace(scaffold.Content)) {
		return shared.GeneratedDocument{}, fmt.Errorf("initial draft (%d chars) shorter than scaffold (%d chars)", len(draft), len(strings.TrimSpace(scaffold.Content)))
	}

	for attempt := 0; attempt <= g.maxRepairs; attempt++ {
		findings := Validate(draft, rubric)
		if len(findings) == 0 {
			return wrapDocument(scaffold.Filename, draft), nil
		}
		if attempt == g.maxRepairs {
			return shared.GeneratedDocument{}, fmt.Errorf("rubric failed after %d repair attempts: %d findings", g.maxRepairs, len(findings))
		}
		repairPrompt := buildRepairPrompt(draft, findings)
		repaired, err := g.llm.Generate(ctx, llm.LLMRequest{
			SystemPrompt:   systemPrompt,
			UserMessage:    repairPrompt,
			Temperature:    g.temperature,
			ResponseFormat: "text",
		})
		if err != nil {
			return shared.GeneratedDocument{}, fmt.Errorf("repair attempt %d: %w", attempt+1, err)
		}
		next := strings.TrimSpace(repaired.Content)
		if next == "" {
			return shared.GeneratedDocument{}, fmt.Errorf("repair attempt %d returned empty content", attempt+1)
		}
		draft = next
	}
	// Unreachable: the loop returns on success or on attempt == maxRepairs.
	return shared.GeneratedDocument{}, errors.New("aigen: repair loop exited unexpectedly")
}

func (g *Generator) buildSystemPrompt(docKey string) string {
	var sb strings.Builder
	sb.WriteString("You are a Principal Software Architect drafting a production-grade engineering document for a real engineering team that will execute against it.\n")
	sb.WriteString("Document type: ")
	sb.WriteString(docKey)
	sb.WriteString(".\n\n")
	sb.WriteString("## Output rules (non-negotiable)\n")
	sb.WriteString("1. Output MUST be valid GitHub-Flavored Markdown. Do NOT wrap the output in code fences. Do NOT prefix with 'Here is the document:' or any preamble.\n")
	sb.WriteString("2. The document MUST be AT LEAST 1000 lines long. Use sub-headings (###, ####), tables, mermaid diagrams, code samples, bulleted lists, and numbered procedures to reach that depth with real content — never filler.\n")
	sb.WriteString("3. Preserve every `## ` top-level heading present in the seed scaffold and keep them in the same order. You MAY add deeper sub-headings; you MAY NOT delete or rename top-level headings.\n")
	sb.WriteString("4. Every top-level section must contain at minimum: a 2–3 paragraph narrative introduction, 2+ sub-sections (###), at least one table OR mermaid diagram OR fenced code block, and a closing 'Implications' or 'Trade-offs' paragraph.\n")
	sb.WriteString("5. Never emit literal placeholder strings such as TBD, TODO, Lorem ipsum, placeholder, '...', 'to be defined', or '<insert ...>'. If a detail is genuinely unknowable, make a reasoned recommendation and label it 'Recommended default:'.\n")
	sb.WriteString("6. Cite numbers (latencies, sizes, throughput, p95, error budgets) wherever you make performance claims. Round to plausible engineering values; never leave a quantity vague.\n")
	sb.WriteString("7. Quality bar: write at the level of a senior staff engineer publishing an internal RFC — explicit trade-offs, concrete component boundaries, named interfaces, schemas, error paths.\n\n")
	sb.WriteString("## Required structural depth for this document type\n")
	sb.WriteString(docSkeletonHint(docKey))
	if composed := g.bundle.Compose(); composed != "" {
		sb.WriteString("\n\n---\n\n")
		sb.WriteString("The following skill bundles encode the engineering standards this document MUST conform to:\n\n")
		sb.WriteString(composed)
	}
	return sb.String()
}

// docSkeletonHint returns a per-document-type structural skeleton hint that
// the AI must use to reach the ≥1000-line depth requirement. Unknown keys
// fall back to a generic guide that still pushes for sub-sections and depth.
func docSkeletonHint(docKey string) string {
	switch docKey {
	case "architecture":
		return `For every top-level section, produce these sub-sections:
- ### Context & Goals  — why this section matters; success criteria
- ### Decisions  — numbered list of architectural decisions with rationale and rejected alternatives
- ### Component breakdown  — table of (Component | Responsibility | Technologies | Dependencies | Owner)
- ### Data contracts  — schema snippets in fenced code blocks (json/sql/go)
- ### Failure modes  — enumerated failure scenarios with detection + mitigation
- ### Mermaid diagram  — sequence or component diagram in a ` + "```mermaid" + ` block
- ### Cross-cutting concerns  — observability, security, capacity planning
- ### Trade-offs & open questions  — explicit list

Section 2 (System Components) must include a per-component sub-section (### N. <Component>) for EVERY component named in the canonical state architecture.components list, each ≥ 200 lines.
Section 4 (Data Flow) must contain at least TWO mermaid diagrams (sequenceDiagram + flowchart).`
	case "roadmap":
		return `For every top-level section, produce these sub-sections:
- ### Objective
- ### Scope (in / out)
- ### Deliverables  — numbered list with acceptance criteria
- ### Exit Criteria  — measurable
- ### Dependencies & blockers
- ### Risks  — table of (Risk | Likelihood | Impact | Mitigation | Owner)
- ### Estimation  — effort, team size, calendar weeks

Section 3 (Phase Breakdown) must contain one ### sub-section per execution_plan entry, each ≥ 100 lines, fully populated with all seven fields above.
Section 4 (Risks) must contain a risk register table with EVERY risk from canonical state, followed by per-risk paragraphs.`
	case "plan":
		return `For every top-level section, produce these sub-sections:
- ### Module charter  — purpose, boundary, public surface
- ### Public interfaces  — function signatures, types, error contracts
- ### Internal design  — algorithm, data structures, complexity
- ### Tests required  — table of (Test name | What it asserts | Inputs | Expected output)
- ### Files to create  — explicit paths
- ### Validation  — commands to run, expected output

Section 4 (Tasks) must contain one ### sub-section per task, each ≥ 80 lines, with all six fields above filled in.`
	case "readme":
		return `For every top-level section, produce these sub-sections:
- ### What & why  — problem framing, value proposition
- ### Architecture at a glance  — mermaid component diagram
- ### Key concepts  — glossary table
- ### Walkthrough  — worked example with code snippets and CLI output
- ### Configuration reference  — table of every env var
- ### Troubleshooting  — numbered failure modes with diagnosis steps
- ### Roadmap pointer  — next-phase summary

Produce a comprehensive Configuration Reference table that includes EVERY env var the system reads.`
	default:
		return `For every top-level section, produce a Context paragraph, 2–4 named sub-sections (###), at least one table or mermaid diagram, concrete code snippets where applicable, and a closing Implications paragraph. Expand each sub-section with worked examples and quantitative detail until total document length reaches 1000+ lines of genuine content.`
	}
}

func buildInitialUserPrompt(key string, scaffold shared.GeneratedDocument, s state.CanonicalState) string {
	var sb strings.Builder
	sb.WriteString("Produce the final `")
	sb.WriteString(key)
	sb.WriteString("` document. Use the deterministic scaffold below as the FACTUAL SEED — every claim it contains must be preserved — then expand it into a production-grade document of AT LEAST 1000 lines, fully populated with sub-sections, tables, mermaid diagrams, code snippets, and concrete examples drawn from the canonical state.\n\n")
	sb.WriteString("Hard requirements:\n")
	sb.WriteString("- ≥ 1000 lines total (not soft-target — hard floor)\n")
	sb.WriteString("- every `## ` heading from the scaffold preserved, in the same order\n")
	sb.WriteString("- every component / risk / assumption / open question / execution_plan entry from the canonical state given its own named sub-section or table row\n")
	sb.WriteString("- zero placeholders (TBD/TODO/Lorem/...); make reasoned 'Recommended default:' calls instead\n")
	sb.WriteString("- at least one mermaid diagram per top-level section where structure matters\n\n")
	sb.WriteString("## Canonical state context\n\n")
	sb.WriteString(summariseState(s))
	sb.WriteString("\n\n## Deterministic scaffold (factual seed — expand, do not summarise)\n\n")
	sb.WriteString(scaffold.Content)
	return sb.String()
}

func buildRepairPrompt(draft string, findings []RubricFinding) string {
	var sb strings.Builder
	needsExpansion := false
	for _, f := range findings {
		if strings.Contains(f.Reason, "minimum is") || strings.Contains(f.Reason, "has ") {
			needsExpansion = true
			break
		}
	}
	sb.WriteString("The previous draft failed the document rubric. Produce a CORRECTED FULL document (not a diff, not a patch) that resolves every finding below.\n\n")
	if needsExpansion {
		sb.WriteString("The primary problem is INSUFFICIENT DEPTH. Do not edit the previous draft sentence-by-sentence — EXPAND it. Add sub-sections, tables, worked examples, mermaid diagrams, and concrete code snippets until each section comfortably exceeds its minimum and the full document exceeds 1000 lines. Preserve every fact already in the draft; never delete content to make room.\n\n")
	}
	sb.WriteString("Findings to resolve:\n\n")
	for _, f := range findings {
		sb.WriteString(f.String())
		sb.WriteString("\n")
	}
	sb.WriteString("\n## Previous draft (expand, do not summarise)\n\n")
	sb.WriteString(draft)
	return sb.String()
}

// summariseState renders a compact, deterministic textual summary of the parts
// of the canonical state that matter for document generation. It does not dump
// raw JSON to keep the prompt token-efficient.
func summariseState(s state.CanonicalState) string {
	var sb strings.Builder
	for _, field := range []string{"title", "problem", "target_users", "value_proposition"} {
		if v := stringFromMap(s.Idea, field); v != "" {
			sb.WriteString(capitaliseLabel(field))
			sb.WriteString(": ")
			sb.WriteString(v)
			sb.WriteString("\n")
		}
	}
	if names := componentNames(s.Architecture); len(names) > 0 {
		sb.WriteString("Architecture components: ")
		sb.WriteString(strings.Join(names, ", "))
		sb.WriteString("\n")
	}
	if len(s.ExecutionPlan) > 0 {
		sb.WriteString(fmt.Sprintf("Execution plan items: %d\n", len(s.ExecutionPlan)))
	}
	if len(s.Risks) > 0 {
		sb.WriteString(fmt.Sprintf("Risks identified: %d\n", len(s.Risks)))
	}
	if len(s.Assumptions) > 0 {
		sb.WriteString(fmt.Sprintf("Assumptions: %d\n", len(s.Assumptions)))
	}
	if len(s.OpenQuestions) > 0 {
		sb.WriteString(fmt.Sprintf("Open questions: %d\n", len(s.OpenQuestions)))
	}
	return sb.String()
}

// stringFromMap returns m[key] as a trimmed string, or "" when absent or not a
// string-typed value.
func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// componentNames extracts the "name" field from each entry under
// architecture.components when present. Returns a deterministically sorted
// slice. Tolerates either []any or []map[string]any.
func componentNames(arch map[string]any) []string {
	if arch == nil {
		return nil
	}
	raw, ok := arch["components"]
	if !ok {
		return nil
	}
	var entries []any
	switch v := raw.(type) {
	case []any:
		entries = v
	case []map[string]any:
		for _, e := range v {
			entries = append(entries, e)
		}
	default:
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		if n := stringFromMap(m, "name"); n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

// wrapDocument builds the GeneratedDocument result with the LineCount derived
// from the final content (newline count + 1, matching the deterministic path).
func wrapDocument(filename, content string) shared.GeneratedDocument {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return shared.GeneratedDocument{
		Filename:  filename,
		Content:   content,
		LineCount: strings.Count(content, "\n"),
		Source:    "ai",
	}
}

// capitaliseLabel transforms "target_users" into "Target users" for display.
func capitaliseLabel(field string) string {
	s := strings.ReplaceAll(field, "_", " ")
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func sortedKeys(m map[string]shared.GeneratedDocument) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
