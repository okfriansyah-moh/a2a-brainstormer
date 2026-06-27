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
	"a2a-brainstorm/backend/internal/platform/config"
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
	llm            llm.LLMProvider
	bundle         SkillBundle
	maxRepairs     int
	temperature    float64
	mode           Mode
	logger         *slog.Logger
	archEnricher   *ArchEnricher
	planEnricher   *PlanEnricher
	readmeEnricher *ReadmeEnricher
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

// SetArchEnricher wires an ArchEnricher into the generator. When set, the
// enricher runs a single LLM pre-pass against the canonical state before the
// "architecture" document is enhanced, populating optional narrative fields.
// Returns the receiver for method chaining.
func (g *Generator) SetArchEnricher(e *ArchEnricher) *Generator {
	g.archEnricher = e
	return g
}

// SetPlanEnricher wires a PlanEnricher into the generator. When set, the
// enricher runs a single LLM pre-pass against the canonical state before the
// "plan" document is enhanced, populating per-phase quality fields.
// Returns the receiver for method chaining.
func (g *Generator) SetPlanEnricher(e *PlanEnricher) *Generator {
	g.planEnricher = e
	return g
}

// SetReadmeEnricher wires a ReadmeEnricher into the generator. When set, the
// enricher runs a single LLM pre-pass against the canonical state before the
// "readme" document is enhanced, populating optional narrative README fields.
// Returns the receiver for method chaining.
func (g *Generator) SetReadmeEnricher(e *ReadmeEnricher) *Generator {
	g.readmeEnricher = e
	return g
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
	return g.EnhanceWithProgress(ctx, s, scaffolds, EnhanceOpts{})
}

// EnhanceWithProgress is Enhance with optional progress and token-streaming callbacks.
// opts.ProgressFn is invoked at key generation stages (enricher, draft, repair, complete).
// opts.TokenFn is invoked for each LLM token when the provider supports streaming.
// When either callback is nil it is silently skipped.
// Callers must not modify opts after passing it — the callbacks are invoked from
// goroutines spawned for each document key.
func (g *Generator) EnhanceWithProgress(ctx context.Context, s state.CanonicalState, scaffolds map[string]shared.GeneratedDocument, opts EnhanceOpts) (map[string]shared.GeneratedDocument, error) {
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

	// Run the architecture enricher pre-pass when applicable — fills optional
	// narrative fields in the state before the "architecture" doc is enhanced.
	enrichedArchState := s
	if _, hasArch := scaffolds["architecture"]; hasArch && g.archEnricher != nil && config.GetArchEnricherEnabled() {
		if opts.ProgressFn != nil {
			opts.ProgressFn("architecture", DocStepEnricher, "Running architecture enricher pre-pass…", ProgressMeta{})
		}
		enrichedArchState, _ = g.archEnricher.Enrich(ctx, s)
	}

	// Run the plan enricher pre-pass when applicable — fills per-phase quality
	// fields in the state before the "plan" doc is enhanced.
	enrichedPlanState := s
	if _, hasPlan := scaffolds["plan"]; hasPlan && g.planEnricher != nil && config.GetPlanEnricherEnabled() {
		if opts.ProgressFn != nil {
			opts.ProgressFn("plan", DocStepEnricher, "Running plan enricher pre-pass…", ProgressMeta{})
		}
		enrichedPlanState, _ = g.planEnricher.Enrich(ctx, s)
	}

	// Run the readme enricher pre-pass when applicable — fills optional README
	// narrative fields in the state before the "readme" doc is enhanced.
	enrichedReadmeState := s
	if _, hasReadme := scaffolds["readme"]; hasReadme && g.readmeEnricher != nil && config.GetReadmeEnricherEnabled() {
		if opts.ProgressFn != nil {
			opts.ProgressFn("readme", DocStepEnricher, "Running readme enricher pre-pass…", ProgressMeta{})
		}
		enrichedReadmeState, _ = g.readmeEnricher.Enrich(ctx, s)
	}

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
		stateForKey := s
		if key == "architecture" {
			stateForKey = enrichedArchState
		} else if key == "plan" {
			stateForKey = enrichedPlanState
		} else if key == "readme" {
			stateForKey = enrichedReadmeState
		}
		go func(k string, scaffold shared.GeneratedDocument, st state.CanonicalState) {
			defer wg.Done()
			enhanced, err := g.enhanceOneWithOpts(ctx, k, scaffold, st, opts)
			mu.Lock()
			results[k] = result{doc: enhanced, err: err}
			mu.Unlock()
		}(key, scaffolds[key], stateForKey)
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

// enhanceOne produces one AI-rewritten document without progress callbacks.
func (g *Generator) enhanceOne(ctx context.Context, key string, scaffold shared.GeneratedDocument, s state.CanonicalState) (shared.GeneratedDocument, error) {
	return g.enhanceOneWithOpts(ctx, key, scaffold, s, EnhanceOpts{})
}

// enhanceOneWithOpts produces one AI-rewritten document, emitting optional
// progress and token-streaming events via opts callbacks.
func (g *Generator) enhanceOneWithOpts(ctx context.Context, key string, scaffold shared.GeneratedDocument, s state.CanonicalState, opts EnhanceOpts) (shared.GeneratedDocument, error) {
	if config.IsSectionSequentialDoc(key) {
		return g.sectionSequentialEnhance(ctx, key, scaffold, s, opts)
	}
	return g.enhanceOneMonolithic(ctx, key, scaffold, s, opts)
}

// enhanceOneMonolithic rewrites the entire document in one LLM pass (legacy path).
func (g *Generator) enhanceOneMonolithic(ctx context.Context, key string, scaffold shared.GeneratedDocument, s state.CanonicalState, opts EnhanceOpts) (shared.GeneratedDocument, error) {
	rubric := RubricFor(key)
	systemPrompt := g.buildSystemPrompt(key)
	userPrompt := buildInitialUserPrompt(key, scaffold, s)

	if opts.ProgressFn != nil {
		opts.ProgressFn(key, DocStepDraft, "Generating first draft with AI…", ProgressMeta{})
	}

	var draft string
	// Use streaming when the provider supports it and a token callback is wired.
	if opts.TokenFn != nil {
		if sp, ok := g.llm.(llm.StreamingLLMProvider); ok {
			d, err := g.generateStreamingDraft(ctx, key, sp, llm.LLMRequest{
				SystemPrompt:   systemPrompt,
				UserMessage:    userPrompt,
				Temperature:    g.temperature,
				ResponseFormat: "text",
			}, opts.TokenFn)
			if err != nil {
				return shared.GeneratedDocument{}, fmt.Errorf("initial generate (stream): %w", err)
			}
			draft = d
		}
	}
	if draft == "" {
		resp, err := g.llm.Generate(ctx, llm.LLMRequest{
			SystemPrompt:   systemPrompt,
			UserMessage:    userPrompt,
			Temperature:    g.temperature,
			ResponseFormat: "text",
		})
		if err != nil {
			return shared.GeneratedDocument{}, fmt.Errorf("initial generate: %w", err)
		}
		draft = strings.TrimSpace(resp.Content)
	}

	if draft == "" {
		return shared.GeneratedDocument{}, errors.New("initial draft was empty")
	}
	// Allow the AI to produce a more concise rewrite — only reject if the draft is
	// less than 50% of the scaffold size, which indicates the LLM stripped content
	// rather than rewrote it. Raw char count is not the quality gate; the rubric is.
	scaffoldLen := len(strings.TrimSpace(scaffold.Content))
	if scaffoldLen > 0 && len(draft) < scaffoldLen/2 {
		return shared.GeneratedDocument{}, fmt.Errorf("initial draft (%d chars) is less than 50%% of scaffold (%d chars) — likely truncated", len(draft), scaffoldLen)
	}

	for attempt := 0; attempt <= g.maxRepairs; attempt++ {
		findings := Validate(draft, rubric)
		if len(findings) == 0 {
			if opts.ProgressFn != nil {
				opts.ProgressFn(key, DocStepComplete, "Document generated successfully.", ProgressMeta{})
			}
			return wrapDocument(scaffold.Filename, draft), nil
		}
		if attempt == g.maxRepairs {
			// Prefer the AI draft over the deterministic scaffold when the LLM
			// produced content but the rubric is still incomplete. Operators see
			// AI-generated output instead of the template fallback badge.
			g.logger.WarnContext(ctx, "aigen_rubric_incomplete",
				slog.String("doc_key", key),
				slog.Int("findings", len(findings)),
				slog.Int("repair_attempts", g.maxRepairs),
			)
			if opts.ProgressFn != nil {
				opts.ProgressFn(key, DocStepComplete, "Document generated (rubric warnings remain).", ProgressMeta{})
			}
			return wrapDocument(scaffold.Filename, draft), nil
		}
		if opts.ProgressFn != nil {
			opts.ProgressFn(key, DocStepRepair, fmt.Sprintf("Repair pass %d/%d — fixing %d rubric finding(s)…", attempt+1, g.maxRepairs, len(findings)), ProgressMeta{})
		}
		repairPrompt := buildRepairPrompt(key, draft, findings)
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

// generateStreamingDraft calls sp.GenerateStream, forwards each token to tokenFn,
// and returns the fully-assembled draft string.
func (g *Generator) generateStreamingDraft(ctx context.Context, key string, sp llm.StreamingLLMProvider, req llm.LLMRequest, tokenFn TokenFunc) (string, error) {
	ch, err := sp.GenerateStream(ctx, req)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			return "", chunk.Err
		}
		if chunk.Text != "" {
			sb.WriteString(chunk.Text)
			tokenFn(key, chunk.Text)
		}
		if chunk.Done {
			break
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

func (g *Generator) buildSystemPrompt(docKey string) string {
	var sb strings.Builder
	if docKey == "readme" {
		sb.WriteString("You are an expert technical writer creating a beginner-friendly product README. Your audience is a developer who has NEVER seen this project — they are asking 'what is this and should I use it?' not 'how do I implement it'.\n")
		sb.WriteString("Document type: readme.\n\n")
		sb.WriteString("## Output rules (non-negotiable)\n")
		sb.WriteString("1. Output MUST be valid GitHub-Flavored Markdown. Do NOT wrap the output in code fences. Do NOT prefix with 'Here is the document:' or any preamble.\n")
		sb.WriteString("2. TARGET LENGTH: 250–500 lines. Concise beats exhaustive. A developer should understand the project in 5 minutes of reading.\n")
		sb.WriteString("3. Preserve every `## ` top-level heading present in the seed scaffold and keep them in the same order.\n")
		sb.WriteString("4. Every section MUST be short: 1–3 sentences of narrative, then the key content (bullet list, code block, or table). NO 2–3 paragraph introductions. NO 'Implications' paragraphs. NO 'Trade-offs' sub-sections.\n")
		sb.WriteString("5. Never emit literal placeholder strings such as TBD, TODO, '<insert ...>', or '<repository-url>'.\n")
		sb.WriteString("6. Write at product-overview level: explain what the project is, why it exists, and how to get started. Do NOT deep-dive into internal implementation details.\n")
		sb.WriteString("7. FORBIDDEN content: configuration tables with 20+ env vars, troubleshooting sections, domain model tables, sequence diagrams, 'For AI Agents' appendix, per-module technical breakdowns, and detailed API documentation. These belong in separate engineering docs, NOT in the README.\n")
		sb.WriteString("8. Do NOT add a `## For AI Agents` appendix. End with the `## Contributing` section.\n\n")
	} else {
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
		sb.WriteString("7. Quality bar: write at the level of a senior staff engineer publishing an internal RFC — explicit trade-offs, concrete component boundaries, named interfaces, schemas, error paths.\n")
		sb.WriteString("8. Dual-audience: every document MUST end with a `## For AI Agents` appendix containing `### Stack`, `### Key Contracts`, `### Implementation Order`, and `### Out of Scope` sub-sections with concrete, pasteable guidance for coding agents.\n\n")
	}
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
		return `CRITICAL naming rule: Implementation steps are called "Task N" (not "Phase N"). Never emit "Phase 0 —", "Phase 1 —", "Phase 0:", "Phase 1:", or any "Phase N" variant — always say "Task N".

For every top-level section, produce these sub-sections:
- ### Context & Goals  — why this section matters; success criteria
- ### Decisions  — numbered list of architectural decisions with rationale and rejected alternatives
- ### Component breakdown  — table of (Component | Responsibility | Technologies | Dependencies | Owner)
- ### Data contracts  — schema snippets in fenced code blocks (json/sql/go)
- ### Failure modes  — enumerated failure scenarios with detection + mitigation
- ### Mermaid diagram  — sequence or component diagram in a ` + "```mermaid" + ` block
- ### Cross-cutting concerns  — observability, security, capacity planning
- ### Trade-offs & open questions  — explicit list

Section 2 (System Components) must include a per-component sub-section (### N. <Component>) for EVERY component named in the canonical state architecture.components list, each ≥ 200 lines.
Section 4 (Data Flow) must contain at least TWO mermaid diagrams (sequenceDiagram + flowchart).
End with ## For AI Agents appendix (Stack, Key Contracts, Implementation Order, Out of Scope).`
	case "plan":
		return `For every top-level section, produce these sub-sections:
- ### Goals detail  — acceptance criteria and success metrics
- ### Milestone summary  — per-task summary with entry/exit conditions
- ### Dependency graph  — ASCII art showing task order and parallelism
- ### Task block  — one ### Task N — {name} per execution plan step; each MUST include **Goal:**, **Layer(s) affected:**, **Files to create:**, **Coding standards:**, **Validation:**, **Invariant check:**, **Prompt context needed:**
- ### Deep knowledge reference  — §8 schemas, algorithms, contracts

CRITICAL naming rule: Every step heading MUST use the format "### Task N — {Name}" (not "Phase N —", not "Phase N:"). If the seed scaffold contains "Phase N" titles, rename them to "Task N" in the output. Never emit "Phase 0", "Phase 1", "Phase 2" or any "Phase N" variant — always say "Task N".

Section 5 (Implementation Tasks) must contain one ### Task N subsection per execution_plan entry with all seven canonical fields.
Section 7 (How to Use This Plan) must explain the task execution protocol.
Section 8 (Deep Knowledge Reference) must include the CanonicalState Go struct skeleton.
End with ## For AI Agents appendix (Stack, Key Contracts, Implementation Order, Out of Scope).`
	case "readme":
		return `TARGET: A beginner-friendly product README. Model it after the best open-source project READMEs: clear, scannable, under 500 lines.

STRUCTURE (follow exactly — preserve every ## heading from the scaffold in the same order):
- Title + one-line tagline
- ## Golden Rule — one sentence stating the core invariant
- ## What it is NOT — 3–5 short bullets (1 sentence each), scope boundaries
- ## When to use — 3–5 numbered scenarios, each with: 1–2 sentence description + a real shell code snippet (not pseudo-code)
- ## Installation — prerequisite list (tool | min version) + 3–5 numbered setup steps
- ## Quick Start — single fenced bash block, 3–5 commands that actually work
- ## Architecture — 1 ASCII art diagram (8–12 lines) + 1 mermaid diagram, NO prose walls
- ## Command Reference — single Markdown table: Command | Description (max 10 rows, core commands only)
- ## Tech Stack — one-line-per-technology table: Technology | Role | Why chosen
- ## Repository Format — brief directory tree (≤ 12 lines) with one-line descriptions
- ## Contributing — 2–4 sentences: what to read, how to run tests, PR process

FORBIDDEN (never include these):
- Configuration tables with 10+ env vars
- Troubleshooting sections
- Domain model / database schema tables
- Detailed sequence diagrams of internal flows
- Module-by-module technical breakdowns
- "For AI Agents" appendix
- Any section titled "Walkthrough" or "Key Concepts"
- Paragraph introductions longer than 2 sentences in any section

TONE: Write as if explaining the project to a smart developer who just found it on GitHub. They need to know: what it does, whether it fits their problem, and how to get started in under 5 minutes.`
	default:
		return `For every top-level section, produce a Context paragraph, 2–4 named sub-sections (###), at least one table or mermaid diagram, concrete code snippets where applicable, and a closing Implications paragraph. Expand each sub-section with worked examples and quantitative detail until total document length reaches 1000+ lines of genuine content.`
	}
}

func buildInitialUserPrompt(key string, scaffold shared.GeneratedDocument, s state.CanonicalState) string {
	var sb strings.Builder
	if key == "readme" {
		sb.WriteString("Produce the final `readme` document. Use the deterministic scaffold below as the FACTUAL SEED — every claim it contains must be preserved — then rewrite it as a concise, beginner-friendly product README.\n\n")
		sb.WriteString("Hard requirements:\n")
		sb.WriteString("- 250–500 lines total — concise beats exhaustive\n")
		sb.WriteString("- every `## ` heading from the scaffold preserved, in the same order\n")
		sb.WriteString("- zero placeholders (TBD/TODO/<insert...>); use the actual project name and tech stack\n")
		sb.WriteString("- Quick Start must contain only real, runnable commands for THIS project\n\n")
	} else {
		sb.WriteString("Produce the final `")
		sb.WriteString(key)
		sb.WriteString("` document. Use the deterministic scaffold below as the FACTUAL SEED — every claim it contains must be preserved — then expand it into a production-grade document of AT LEAST 1000 lines, fully populated with sub-sections, tables, mermaid diagrams, code snippets, and concrete examples drawn from the canonical state.\n\n")
		sb.WriteString("Hard requirements:\n")
		sb.WriteString("- ≥ 1000 lines total (not soft-target — hard floor)\n")
		sb.WriteString("- every `## ` heading from the scaffold preserved, in the same order\n")
		sb.WriteString("- every component / risk / assumption / open question / execution_plan entry from the canonical state given its own named sub-section or table row\n")
		sb.WriteString("- zero placeholders (TBD/TODO/Lorem/...); make reasoned 'Recommended default:' calls instead\n")
		sb.WriteString("- at least one mermaid diagram per top-level section where structure matters\n\n")
	}
	sb.WriteString("## Canonical state context\n\n")
	sb.WriteString(summariseState(s))
	sb.WriteString("\n\n## Deterministic scaffold (factual seed — expand, do not summarise)\n\n")
	sb.WriteString(scaffold.Content)
	return sb.String()
}

func buildRepairPrompt(key, draft string, findings []RubricFinding) string {
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
		if key == "readme" {
			sb.WriteString("The primary problem is MISSING or TOO-SHORT SECTIONS. Add the required content to each flagged section while keeping the total document within 250–500 lines. Never add engineering-deep sections (config tables, troubleshooting, domain models) to fix a length finding.\n\n")
		} else {
			sb.WriteString("The primary problem is INSUFFICIENT DEPTH. Do not edit the previous draft sentence-by-sentence — EXPAND it. Add sub-sections, tables, worked examples, mermaid diagrams, and concrete code snippets until each section comfortably exceeds its minimum and the full document exceeds 1000 lines. Preserve every fact already in the draft; never delete content to make room.\n\n")
		}
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
