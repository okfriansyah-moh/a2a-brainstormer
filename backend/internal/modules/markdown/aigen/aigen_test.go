// Package aigen — tests covering skill loading, rubric validation, and the
// AI generator's auto-repair / fallback semantics.
package aigen_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"

	"a2a-brainstorm/backend/internal/modules/markdown/aigen"
	"a2a-brainstorm/backend/internal/modules/state"
	"a2a-brainstorm/backend/internal/platform/llm"
	"a2a-brainstorm/backend/internal/shared"
)

// stubProvider is a deterministic LLMProvider used to exercise the generator's
// repair and fallback paths without any real network I/O.
type stubProvider struct {
	responses []string
	errs      []error
	calls     int
}

func (s *stubProvider) Generate(ctx context.Context, req llm.LLMRequest) (llm.LLMResponse, error) {
	i := s.calls
	s.calls++
	if i < len(s.errs) && s.errs[i] != nil {
		return llm.LLMResponse{}, s.errs[i]
	}
	if i >= len(s.responses) {
		return llm.LLMResponse{}, errors.New("stub: no more responses")
	}
	return llm.LLMResponse{Content: s.responses[i], FinishReason: "stop"}, nil
}

func newBundleFS(t *testing.T) (fstest.MapFS, []string) {
	t.Helper()
	fsys := fstest.MapFS{
		".github/skills/alpha/SKILL.md": {Data: []byte("---\nname: alpha\n---\nAlpha guidance body.\n")},
		".github/skills/beta/SKILL.md":  {Data: []byte("Beta has no frontmatter.\n")},
	}
	return fsys, []string{".github/skills/alpha/SKILL.md", ".github/skills/beta/SKILL.md"}
}

func TestLoadBundle_StripsFrontmatter(t *testing.T) {
	fsys, paths := newBundleFS(t)
	bundle, err := aigen.LoadBundle(fsys, paths)
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	if len(bundle.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(bundle.Skills))
	}
	if got := bundle.Skills[0].Name; got != "alpha" {
		t.Errorf("skill[0].Name = %q, want %q", got, "alpha")
	}
	if strings.Contains(bundle.Skills[0].Prompt, "name: alpha") {
		t.Errorf("frontmatter not stripped: %q", bundle.Skills[0].Prompt)
	}
	if !strings.Contains(bundle.Skills[0].Prompt, "Alpha guidance body") {
		t.Errorf("body missing: %q", bundle.Skills[0].Prompt)
	}
}

func TestLoadBundle_MissingFileErrors(t *testing.T) {
	fsys, _ := newBundleFS(t)
	_, err := aigen.LoadBundle(fsys, []string{".github/skills/missing/SKILL.md"})
	if err == nil {
		t.Fatal("expected error for missing skill, got nil")
	}
}

func TestLoadBundle_EmptyPathsErrors(t *testing.T) {
	if _, err := aigen.LoadBundle(fstest.MapFS{}, nil); err == nil {
		t.Fatal("expected error for empty paths, got nil")
	}
}

func TestBundle_Compose_OrderStable(t *testing.T) {
	fsys, paths := newBundleFS(t)
	bundle, _ := aigen.LoadBundle(fsys, paths)
	composed := bundle.Compose()
	if !strings.Contains(composed, "## Skill: alpha") || !strings.Contains(composed, "## Skill: beta") {
		t.Fatalf("composed missing skill headers: %q", composed)
	}
	if idxAlpha, idxBeta := strings.Index(composed, "alpha"), strings.Index(composed, "beta"); idxAlpha > idxBeta {
		t.Errorf("expected alpha before beta; alpha=%d beta=%d", idxAlpha, idxBeta)
	}
}

func TestValidate_PassesGoodDocument(t *testing.T) {
	// Custom rubric that mirrors the readme section structure but without the
	// production MinTotalLines=1000 floor — this test exercises Validate's
	// per-section logic, not the doc-level depth contract.
	custom := aigen.Rubric{DocKey: "readme-test", Sections: []aigen.SectionRule{
		{Heading: "Overview", MinChars: 100},
		{Heading: "Architecture", MinChars: 100},
		{Heading: "Roadmap", MinChars: 100},
		{Heading: "Getting Started", MinChars: 100},
	}}
	content := "# Title\n\n## Overview\n\n" + strings.Repeat("ok ", 200) + "\n\n## Architecture\n\n" + strings.Repeat("good ", 200) + "\n\n## Roadmap\n\n" + strings.Repeat("plan ", 200) + "\n\n## Getting Started\n\n" + strings.Repeat("go ", 200) + "\n"
	findings := aigen.Validate(content, custom)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d: %+v", len(findings), findings)
	}
}

func TestValidate_FlagsTotalLineFloor(t *testing.T) {
	r := aigen.Rubric{DocKey: "x", MinTotalLines: 100}
	content := "## Overview\n\nshort\n"
	findings := aigen.Validate(content, r)
	found := false
	for _, f := range findings {
		if strings.Contains(f.Reason, "document has") && strings.Contains(f.Reason, "lines") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected MinTotalLines finding; got %+v", findings)
	}
}

func TestValidate_FlagsTotalCharFloor(t *testing.T) {
	r := aigen.Rubric{DocKey: "x", MinTotalChars: 5000}
	content := "## Overview\n\nshort body\n"
	findings := aigen.Validate(content, r)
	found := false
	for _, f := range findings {
		if strings.Contains(f.Reason, "document has") && strings.Contains(f.Reason, "chars") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected MinTotalChars finding; got %+v", findings)
	}
}

func TestValidate_FlagsMissingHeading(t *testing.T) {
	content := "# x\n\n## Overview\n\n" + strings.Repeat("ok ", 200) + "\n"
	r := aigen.Rubric{DocKey: "x", Sections: []aigen.SectionRule{
		{Heading: "Overview", MinChars: 100},
		{Heading: "Architecture", MinChars: 100},
	}}
	findings := aigen.Validate(content, r)
	if len(findings) == 0 {
		t.Fatal("expected findings for missing headings")
	}
}

func TestValidate_FlagsPlaceholders(t *testing.T) {
	content := "## Overview\n\nTODO: write this section." + strings.Repeat(" body", 200) + "\n"
	r := aigen.Rubric{DocKey: "x", Sections: []aigen.SectionRule{{Heading: "Overview", MinChars: 100}}}
	findings := aigen.Validate(content, r)
	found := false
	for _, f := range findings {
		if strings.Contains(f.Reason, "placeholder") || strings.Contains(f.Reason, "TODO") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected placeholder finding; got %+v", findings)
	}
}

func TestValidate_FlagsMissingKeyword(t *testing.T) {
	body := strings.Repeat("text ", 200)
	content := "## Data Flow\n\n" + body + "\n"
	r := aigen.Rubric{DocKey: "x", Sections: []aigen.SectionRule{{Heading: "Data Flow", MinChars: 100, RequiredKeywords: []string{"```mermaid"}}}}
	findings := aigen.Validate(content, r)
	if len(findings) == 0 {
		t.Fatal("expected finding for missing mermaid block")
	}
}

func newReadmeScaffold() shared.GeneratedDocument {
	body := "## Overview\n\nseed.\n\n## Architecture\n\nseed.\n\n## Roadmap\n\nseed.\n\n## Getting Started\n\nseed.\n"
	return shared.GeneratedDocument{Filename: "0001-readme.md", Content: body, LineCount: 9}
}

// custom-readme is a docKey not present in defaultRubrics; it triggers an
// empty rubric (trivially passes Validate) so generator-mechanics tests are
// not coupled to the production line-count floor.
const testDocKey = "custom-readme"

func newPassingReadme() string {
	// Pad each section heavily so the document satisfies the production
	// readme rubric (per-section MinChars ≥ 1000, MinTotalLines ≥ 1000,
	// MinTotalChars ≥ 35000). The repeated phrase contains no forbidden
	// placeholder tokens.
	pad := strings.Repeat("substantive body paragraph line with real content.\n", 280)
	return "# Readme\n\n## Overview\n\n" + pad + "\n## Architecture\n\n" + pad + "\n## Roadmap\n\n" + pad + "\n## Getting Started\n\n" + pad + "\n"
}

func TestGenerator_Enhance_HappyPath(t *testing.T) {
	bundle := aigen.SkillBundle{Skills: []aigen.Skill{{Name: "x", Prompt: "be precise"}}}
	stub := &stubProvider{responses: []string{newPassingReadme()}}
	g := aigen.New(stub, bundle, 2, 0.2, aigen.ModeHybrid, slog.Default())

	scaffolds := map[string]shared.GeneratedDocument{testDocKey: newReadmeScaffold()}
	out, err := g.Enhance(context.Background(), state.CanonicalState{}, scaffolds)
	if err != nil {
		t.Fatalf("Enhance err: %v", err)
	}
	if !strings.Contains(out[testDocKey].Content, "## Overview") {
		t.Errorf("enhanced content missing Overview")
	}
	if out[testDocKey].Content == scaffolds[testDocKey].Content {
		t.Errorf("expected enhanced content to differ from scaffold")
	}
	if stub.calls != 1 {
		t.Errorf("expected 1 LLM call, got %d", stub.calls)
	}
}

func TestGenerator_Enhance_RepairThenSuccess(t *testing.T) {
	bundle := aigen.SkillBundle{Skills: []aigen.Skill{{Name: "x", Prompt: "p"}}}
	// Use the production "readme" key so the first short response fails the
	// rubric, triggers a repair, then the second long response passes the
	// per-section rules (we do not assert MinTotalLines here — it is exercised
	// in TestValidate_FlagsTotalLineFloor).
	long := newPassingReadme() + strings.Repeat("\nfiller line ", 4000)
	short := "# r\n\n## Overview\n\n" + strings.Repeat("a ", 200) + "\n\n## Architecture\n\nshort\n\n## Roadmap\n\n" + strings.Repeat("b ", 200) + "\n\n## Getting Started\n\n" + strings.Repeat("c ", 200) + "\n"
	stub := &stubProvider{responses: []string{short, long}}
	g := aigen.New(stub, bundle, 2, 0.2, aigen.ModeHybrid, slog.Default())

	scaffolds := map[string]shared.GeneratedDocument{"readme": newReadmeScaffold()}
	out, err := g.Enhance(context.Background(), state.CanonicalState{}, scaffolds)
	if err != nil {
		t.Fatalf("Enhance err: %v", err)
	}
	if stub.calls != 2 {
		t.Errorf("expected 2 LLM calls (initial+1 repair), got %d", stub.calls)
	}
	if !strings.Contains(out["readme"].Content, "Architecture") {
		t.Errorf("missing Architecture in final content")
	}
}

func TestGenerator_Enhance_HybridFallbackOnLLMError(t *testing.T) {
	bundle := aigen.SkillBundle{Skills: []aigen.Skill{{Name: "x", Prompt: "p"}}}
	stub := &stubProvider{errs: []error{errors.New("boom")}}
	g := aigen.New(stub, bundle, 2, 0.2, aigen.ModeHybrid, slog.Default())

	scaffold := newReadmeScaffold()
	out, err := g.Enhance(context.Background(), state.CanonicalState{}, map[string]shared.GeneratedDocument{testDocKey: scaffold})
	if err != nil {
		t.Fatalf("hybrid mode must not error on LLM failure: %v", err)
	}
	if out[testDocKey].Content != scaffold.Content {
		t.Errorf("expected fallback to scaffold; got different content")
	}
}

func TestGenerator_Enhance_AIModeReturnsError(t *testing.T) {
	bundle := aigen.SkillBundle{Skills: []aigen.Skill{{Name: "x", Prompt: "p"}}}
	stub := &stubProvider{errs: []error{errors.New("boom")}}
	g := aigen.New(stub, bundle, 2, 0.2, aigen.ModeAI, slog.Default())

	_, err := g.Enhance(context.Background(), state.CanonicalState{}, map[string]shared.GeneratedDocument{testDocKey: newReadmeScaffold()})
	if err == nil {
		t.Fatal("expected error in AI mode")
	}
}

func TestGenerator_Enhance_EmptyBundleFallsBack(t *testing.T) {
	stub := &stubProvider{}
	g := aigen.New(stub, aigen.SkillBundle{}, 2, 0.2, aigen.ModeHybrid, slog.Default())

	scaffold := newReadmeScaffold()
	out, err := g.Enhance(context.Background(), state.CanonicalState{}, map[string]shared.GeneratedDocument{testDocKey: scaffold})
	if err != nil {
		t.Fatalf("Enhance err: %v", err)
	}
	if out[testDocKey].Content != scaffold.Content {
		t.Errorf("expected scaffold passthrough on empty bundle")
	}
	if stub.calls != 0 {
		t.Errorf("expected zero LLM calls with empty bundle, got %d", stub.calls)
	}
}
