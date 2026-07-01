package aigen

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/llm"
	"a2a-brainstorm/backend/internal/shared"
)

// sectionSequentialEnhance enhances one document section-by-section then merges.
func (g *Generator) sectionSequentialEnhance(ctx context.Context, key string, scaffold shared.GeneratedDocument, s state.CanonicalState, opts EnhanceOpts) (shared.GeneratedDocument, error) {
	rubric := RubricFor(key)
	buckets, err := BucketScaffoldSections(scaffold.Content, rubric)
	if err != nil {
		return shared.GeneratedDocument{}, fmt.Errorf("section sequential: bucket: %w", err)
	}

	preamble, _ := ExtractPreamble(scaffold.Content, rubric.Sections[0].Heading)

	enhanced := make([]EnhancedSection, 0, len(buckets))
	anyFallback := false
	maxPrior := config.GetAIGenPriorSectionMaxChars()

	for i, bucket := range buckets {
		rule := rubric.Sections[i]
		prior := SummarizePriorSections(enhanced, maxPrior)

		opts.EmitProgress(key, DocStepSectionEnhance, fmt.Sprintf("Enhancing §%s…", rule.Heading), ProgressMeta{
			Section: rule.Heading,
		})

		body, fallback, err := g.enhanceSectionBody(ctx, key, rule, bucket, prior, s, opts)
		if err != nil {
			return shared.GeneratedDocument{}, err
		}
		if fallback {
			anyFallback = true
			g.logger.WarnContext(ctx, "aigen_section_fallback",
				slog.String("doc_key", key),
				slog.String("section", rule.Heading),
			)
			body = bucket.Body
		}

		enhanced = append(enhanced, EnhancedSection{
			Heading:     rule.Heading,
			HeadingLine: bucket.HeadingLine,
			Body:        body,
		})
	}

	merged := MergeSections(preamble, enhanced)
	merged, err = g.repairDocumentSections(ctx, key, merged, rubric, s, opts)
	if err != nil {
		return shared.GeneratedDocument{}, err
	}

	if config.GetAIGenCoherenceEnabled() {
		coherent, err := g.runCoherencePass(ctx, key, merged, rubric, s, opts)
		if err != nil && g.mode == ModeAI {
			return shared.GeneratedDocument{}, fmt.Errorf("coherence: %w", err)
		}
		if err == nil {
			merged = coherent
		}
	}

	opts.EmitProgress(key, DocStepComplete, "Document generated successfully.", ProgressMeta{})

	doc := wrapDocument(scaffold.Filename, merged)
	if anyFallback {
		doc.Source = "ai_fallback"
	}
	return doc, nil
}

func (g *Generator) enhanceSectionBody(ctx context.Context, docKey string, rule SectionRule, bucket EnhancedSection, priorSummary string, s state.CanonicalState, opts EnhanceOpts) (body string, fallback bool, err error) {
	systemPrompt := g.buildSectionSystemPrompt(docKey, rule, s)
	userPrompt := buildSectionUserPrompt(docKey, bucket, priorSummary, s, rule)

	resp, err := g.llm.Generate(ctx, llm.LLMRequest{
		SystemPrompt:   systemPrompt,
		UserMessage:    userPrompt,
		Temperature:    g.temperature,
		ResponseFormat: "text",
	})
	if err != nil {
		if g.mode == ModeAI {
			return "", false, fmt.Errorf("section enhance %q: %w", rule.Heading, err)
		}
		return "", true, nil
	}
	draft := strings.TrimSpace(resp.Content)
	if draft == "" {
		if g.mode == ModeAI {
			return "", false, fmt.Errorf("section enhance %q: empty draft", rule.Heading)
		}
		return "", true, nil
	}

	for attempt := 0; attempt <= g.maxRepairs; attempt++ {
		findings := ValidateSection(draft, rule)
		if len(findings) == 0 {
			return draft, false, nil
		}
		if attempt == g.maxRepairs {
			return draft, false, nil
		}
		opts.EmitProgress(docKey, DocStepSectionRepair, fmt.Sprintf("Repairing §%s (%d finding(s))…", rule.Heading, len(findings)), ProgressMeta{
			Section: rule.Heading,
		})
		repairPrompt := buildSectionRepairPrompt(docKey, rule.Heading, draft, findings)
		repaired, err := g.llm.Generate(ctx, llm.LLMRequest{
			SystemPrompt:   systemPrompt,
			UserMessage:    repairPrompt,
			Temperature:    g.temperature,
			ResponseFormat: "text",
		})
		if err != nil {
			if g.mode == ModeAI {
				return "", false, fmt.Errorf("section repair %q: %w", rule.Heading, err)
			}
			return "", true, nil
		}
		next := strings.TrimSpace(repaired.Content)
		if next == "" {
			if g.mode == ModeAI {
				return "", false, fmt.Errorf("section repair %q: empty", rule.Heading)
			}
			return "", true, nil
		}
		draft = next
	}
	return draft, false, nil
}

func (g *Generator) repairDocumentSections(ctx context.Context, docKey, merged string, rubric Rubric, s state.CanonicalState, opts EnhanceOpts) (string, error) {
	draft := merged
	for attempt := 0; attempt <= g.maxRepairs; attempt++ {
		findings := Validate(draft, rubric)
		if len(findings) == 0 {
			return draft, nil
		}
		if attempt == g.maxRepairs {
			g.logger.WarnContext(ctx, "aigen_rubric_incomplete",
				slog.String("doc_key", docKey),
				slog.Int("findings", len(findings)),
			)
			return draft, nil
		}
		// Repair only document-level findings via full merged repair for those;
		// section-scoped findings get per-section repair.
		sectionFindings := filterSectionFindings(findings)
		if len(sectionFindings) == 0 {
			g.logger.WarnContext(ctx, "aigen_doc_level_findings_only",
				slog.String("doc_key", docKey),
				slog.Int("findings", len(findings)),
			)
			return draft, nil
		}
		for _, f := range sectionFindings {
			rule := findSectionRule(rubric, f.Heading)
			if rule == nil {
				continue
			}
			body, ok := ExtractSectionBody(draft, f.Heading)
			if !ok {
				continue
			}
			opts.EmitProgress(docKey, DocStepSectionRepair, fmt.Sprintf("Document repair §%s…", f.Heading), ProgressMeta{
				Section: f.Heading,
			})
			systemPrompt := g.buildSectionSystemPrompt(docKey, *rule, s)
			repairPrompt := buildSectionRepairPrompt(docKey, f.Heading, body, []RubricFinding{f})
			repaired, err := g.llm.Generate(ctx, llm.LLMRequest{
				SystemPrompt:   systemPrompt,
				UserMessage:    repairPrompt,
				Temperature:    g.temperature,
				ResponseFormat: "text",
			})
			if err != nil {
				if g.mode == ModeAI {
					return "", err
				}
				continue
			}
			next := strings.TrimSpace(repaired.Content)
			if next == "" {
				continue
			}
			replaced, err := ReplaceSectionBody(draft, f.Heading, next)
			if err == nil {
				draft = replaced
			}
		}
	}
	return draft, nil
}

func filterSectionFindings(findings []RubricFinding) []RubricFinding {
	out := make([]RubricFinding, 0)
	for _, f := range findings {
		if f.Heading != "<document>" {
			out = append(out, f)
		}
	}
	return out
}

func findSectionRule(rubric Rubric, heading string) *SectionRule {
	for i := range rubric.Sections {
		if rubric.Sections[i].Heading == heading {
			return &rubric.Sections[i]
		}
	}
	return nil
}

func (g *Generator) buildSectionSystemPrompt(docKey string, rule SectionRule, s state.CanonicalState) string {
	base := g.buildSystemPrompt(docKey, s)
	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n## Section-only output\n")
	sb.WriteString("You are enhancing ONE section only. Output the section BODY only — do NOT include the ## heading line.\n")
	sb.WriteString("Section: ")
	sb.WriteString(rule.Heading)
	sb.WriteString("\nMinimum body length: ")
	sb.WriteString(fmt.Sprintf("%d characters.\n", rule.MinChars))
	return sb.String()
}

func buildSectionUserPrompt(docKey string, bucket EnhancedSection, priorSummary string, s state.CanonicalState, rule SectionRule) string {
	var sb strings.Builder
	sb.WriteString("Enhance section `")
	sb.WriteString(rule.Heading)
	sb.WriteString("` of the `")
	sb.WriteString(docKey)
	sb.WriteString("` document. Expand the scaffold slice into production-grade depth with sub-headings (###), tables, and diagrams as appropriate.\n")
	sb.WriteString("Stay focused on the USER'S PRODUCT from canonical state — do not write about the A2A Brainstorm tool or generic multi-agent frameworks.\n\n")
	sb.WriteString("## Canonical state context\n\n")
	sb.WriteString(summariseState(s))
	if priorSummary != "" {
		sb.WriteString("\n\n## Prior enhanced sections (summary)\n\n")
		sb.WriteString(priorSummary)
	}
	sb.WriteString("\n\n## Deterministic scaffold slice for this section\n\n")
	if bucket.HeadingLine != "" {
		sb.WriteString(bucket.HeadingLine)
		sb.WriteString("\n\n")
	}
	sb.WriteString(bucket.Body)
	return sb.String()
}

func buildSectionRepairPrompt(docKey, heading, draft string, findings []RubricFinding) string {
	var sb strings.Builder
	sb.WriteString("The section body failed the rubric. Produce a CORRECTED section body only (no ## heading) for `")
	sb.WriteString(docKey)
	sb.WriteString("` section `")
	sb.WriteString(heading)
	sb.WriteString("`.\n\nFindings:\n")
	for _, f := range findings {
		sb.WriteString(f.String())
		sb.WriteString("\n")
	}
	sb.WriteString("\n## Previous section body\n\n")
	sb.WriteString(draft)
	return sb.String()
}
