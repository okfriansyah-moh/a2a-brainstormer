package aigen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"a2a-brainstorm/backend/internal/platform/config"
	"a2a-brainstorm/backend/internal/platform/llm"
)

const immutableCoherenceSection = "For AI Agents"

// CoherenceFinding is one cross-section inconsistency from the audit pass.
type CoherenceFinding struct {
	SectionHeading string `json:"section_heading"`
	Issue          string `json:"issue"`
	FixType        string `json:"fix_type"`
	Confidence     string `json:"confidence"`
}

type coherenceAuditResponse struct {
	Findings []CoherenceFinding `json:"findings"`
}

// runCoherencePass executes always-on audit and guarded micro-fixes.
func (g *Generator) runCoherencePass(ctx context.Context, docKey, merged string, rubric Rubric, opts EnhanceOpts) (string, error) {
	if g == nil || g.llm == nil {
		return merged, nil
	}

	opts.EmitProgress(docKey, DocStepCoherenceAudit, "Checking cross-section consistency…", ProgressMeta{})

	findings, err := g.auditCoherence(ctx, docKey, merged)
	if err != nil {
		g.logger.WarnContext(ctx, "aigen_coherence_skip",
			slog.String("doc_key", docKey),
			slog.String("reason", err.Error()),
		)
		return merged, nil
	}

	opts.EmitProgress(docKey, DocStepCoherenceAudit, "Consistency audit complete.", ProgressMeta{
		FindingsCount: len(findings),
	})

	if len(findings) == 0 {
		return merged, nil
	}

	before := merged
	after := merged
	contradictionHeadings := make(map[string]bool)
	for _, f := range findings {
		if strings.Contains(f.SectionHeading, immutableCoherenceSection) {
			g.logger.InfoContext(ctx, "aigen_coherence_immutable_skip",
				slog.String("doc_key", docKey),
				slog.String("section", f.SectionHeading),
			)
			continue
		}
		if f.FixType == "contradiction" {
			contradictionHeadings[f.SectionHeading] = true
		}
		opts.EmitProgress(docKey, DocStepCoherenceFix, f.Issue, ProgressMeta{Section: f.SectionHeading})
		body, ok := ExtractSectionBody(after, f.SectionHeading)
		if !ok {
			continue
		}
		fixed, err := g.microFixCoherence(ctx, docKey, f.SectionHeading, body, f)
		if err != nil {
			g.logger.WarnContext(ctx, "aigen_coherence_microfix_failed",
				slog.String("doc_key", docKey),
				slog.String("section", f.SectionHeading),
				slog.Any("error", err),
			)
			continue
		}
		replaced, err := ReplaceSectionBody(after, f.SectionHeading, fixed)
		if err != nil {
			continue
		}
		after = replaced
	}

	guarded, err := ApplyCoherenceGuardrails(before, after, rubric, contradictionHeadings)
	if err != nil {
		g.logger.WarnContext(ctx, "aigen_coherence_discarded",
			slog.String("doc_key", docKey),
			slog.String("reason", err.Error()),
		)
		return before, nil
	}
	return guarded, nil
}

func (g *Generator) auditCoherence(ctx context.Context, docKey, merged string) ([]CoherenceFinding, error) {
	system := "You are a document auditor. Return ONLY valid JSON with shape {\"findings\":[{...}]}. Do not rewrite the document."
	user := fmt.Sprintf(`Audit this %s document for cross-section inconsistencies only (terminology, naming, contradictions, cross-references).
Return JSON: {"findings":[{"section_heading":"...","issue":"...","fix_type":"terminology|cross_reference|contradiction|naming","confidence":"high|medium|low"}]}
If no issues, return {"findings":[]}.

Document:
%s`, docKey, merged)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := g.llm.Generate(ctx, llm.LLMRequest{
			SystemPrompt:   system,
			UserMessage:    user,
			Temperature:    0,
			ResponseFormat: "json",
		})
		if err != nil {
			return nil, err
		}
		findings, err := parseCoherenceAudit(resp.Content)
		if err == nil {
			return findings, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func parseCoherenceAudit(content string) ([]CoherenceFinding, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var parsed coherenceAuditResponse
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("parse coherence audit: %w", err)
	}
	if parsed.Findings == nil {
		return nil, errors.New("parse coherence audit: missing findings array")
	}
	return parsed.Findings, nil
}

func (g *Generator) microFixCoherence(ctx context.Context, docKey, heading, body string, finding CoherenceFinding) (string, error) {
	system := g.buildSystemPrompt(docKey) + "\n\nYou fix ONLY the identified inconsistency. Output section body only (no ## heading). Do not shorten tables or code blocks."
	user := fmt.Sprintf(`Section: %q
Issue: %s
Fix type: %s

Fix ONLY this inconsistency. Preserve structure and length. Do not remove content.

Current section body:
%s`, heading, finding.Issue, finding.FixType, body)

	resp, err := g.llm.Generate(ctx, llm.LLMRequest{
		SystemPrompt:   system,
		UserMessage:    user,
		Temperature:    g.temperature,
		ResponseFormat: "text",
	})
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(resp.Content)
	if out == "" {
		return "", errors.New("coherence micro-fix returned empty body")
	}
	return out, nil
}

// ApplyCoherenceGuardrails reverts sections that violate guardrails.
func ApplyCoherenceGuardrails(before, after string, rubric Rubric, contradictionHeadings map[string]bool) (string, error) {
	if !headingsEqual(before, after) {
		return before, errors.New("structural freeze: headings changed")
	}

	minRatio := config.GetAIGenCoherenceMinRatio()
	maxEdit := config.GetAIGenCoherenceMaxEditRatio()

	result := after
	for _, rule := range rubric.Sections {
		beforeBody, okB := ExtractSectionBody(before, rule.Heading)
		afterBody, okA := ExtractSectionBody(result, rule.Heading)
		if !okB || !okA {
			continue
		}
		bTrim := strings.TrimSpace(beforeBody)
		aTrim := strings.TrimSpace(afterBody)
		if len(bTrim) == 0 {
			continue
		}
		if float64(len(aTrim)) < float64(len(bTrim))*minRatio {
			replaced, err := ReplaceSectionBody(result, rule.Heading, beforeBody)
			if err == nil {
				result = replaced
			}
			continue
		}
		waiveEdit := contradictionHeadings != nil && contradictionHeadings[rule.Heading]
		if !waiveEdit && editRatio(bTrim, aTrim) > maxEdit {
			replaced, err := ReplaceSectionBody(result, rule.Heading, beforeBody)
			if err == nil {
				result = replaced
			}
			continue
		}
		beforeFindings := ValidateSection(beforeBody, rule)
		afterFindings := ValidateSection(aTrim, rule)
		if len(beforeFindings) == 0 && len(afterFindings) > 0 {
			replaced, err := ReplaceSectionBody(result, rule.Heading, beforeBody)
			if err == nil {
				result = replaced
			}
		}
	}
	return result, nil
}

func headingsEqual(a, b string) bool {
	ha := ListHeadings(a)
	hb := ListHeadings(b)
	if len(ha) != len(hb) {
		return false
	}
	for i := range ha {
		if ha[i].Title != hb[i].Title {
			return false
		}
	}
	return true
}

func editRatio(before, after string) float64 {
	if len(before) == 0 {
		return 0
	}
	diff := math.Abs(float64(len(after) - len(before)))
	return diff / float64(len(before))
}

// ParseCoherenceAuditForTest exposes parseCoherenceAudit for unit tests.
func ParseCoherenceAuditForTest(content string) ([]CoherenceFinding, error) {
	return parseCoherenceAudit(content)
}
