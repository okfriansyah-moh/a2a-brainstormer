package aigen_test

import (
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/markdown/aigen"
)

func TestExtractPreamble(t *testing.T) {
	scaffold := "# Title\n\n> meta\n\n## 1. Problem Statement\n\nbody\n"
	pre, ok := aigen.ExtractPreamble(scaffold, "1. Problem Statement")
	if !ok {
		t.Fatal("expected preamble")
	}
	if !strings.Contains(pre, "# Title") {
		t.Fatalf("preamble missing title: %q", pre)
	}
	if strings.Contains(pre, "Problem Statement") {
		t.Fatalf("preamble should not include first section: %q", pre)
	}
}

func TestBucketScaffoldSections_PlanIntermediates(t *testing.T) {
	scaffold := strings.Join([]string{
		"# Plan — Implementation Plan",
		"",
		"## 1. Goals",
		"goals body",
		"",
		"## 2. Milestones",
		"milestones body",
		"",
		"## 3. Assumptions",
		"assumptions body",
		"",
		"## 4. Dependency Graph",
		"graph body",
		"",
		"## 5. Implementation Tasks",
		"tasks body",
		"",
		"## 7. How to Use This Plan",
		"howto body",
		"",
		"## 8. Deep Knowledge Reference",
		"dk body",
		"",
		"## For AI Agents",
		"agents body",
	}, "\n")

	rubric := aigen.RubricFor("plan")
	buckets, err := aigen.BucketScaffoldSections(scaffold, rubric)
	if err != nil {
		t.Fatalf("BucketScaffoldSections: %v", err)
	}
	if len(buckets) != 5 {
		t.Fatalf("expected 5 buckets, got %d", len(buckets))
	}
	if !strings.Contains(buckets[0].Body, "goals body") {
		t.Errorf("goals bucket wrong: %q", buckets[0].Body)
	}
	taskBody := buckets[1].Body
	for _, want := range []string{"milestones body", "assumptions body", "graph body", "tasks body"} {
		if !strings.Contains(taskBody, want) {
			t.Errorf("task bucket missing %q: %q", want, taskBody)
		}
	}
}

func TestMergeSections_PreservesOrder(t *testing.T) {
	sections := []aigen.EnhancedSection{
		{Heading: "1. A", HeadingLine: "## 1. A", Body: "aaa"},
		{Heading: "2. B", HeadingLine: "## 2. B", Body: "bbb"},
	}
	merged := aigen.MergeSections("# Title\n\n", sections)
	if !strings.Contains(merged, "## 1. A") || !strings.Contains(merged, "## 2. B") {
		t.Fatalf("headings missing: %q", merged)
	}
	headings := aigen.ListHeadings(merged)
	if len(headings) < 2 {
		t.Fatalf("expected headings in merged doc")
	}
}

func TestSummarizePriorSections_Truncates(t *testing.T) {
	sections := []aigen.EnhancedSection{
		{Heading: "1. A", Body: strings.Repeat("word ", 500)},
		{Heading: "2. B", Body: strings.Repeat("other ", 500)},
	}
	summary := aigen.SummarizePriorSections(sections, 200)
	if len(summary) > 200 {
		t.Fatalf("summary too long: %d", len(summary))
	}
}
