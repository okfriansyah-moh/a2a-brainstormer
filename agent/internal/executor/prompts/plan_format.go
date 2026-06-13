// Package prompts — Approach C prompt injection for plan output format (§8.31.3).
//
// This package must NOT import any backend/internal/ packages. It is a
// self-contained constant + helper used by the agent executor to inject the
// plan task format when the payload's OutputDocs includes "plan".
package prompts

import "strings"

// PlanTaskFormat is the canonical plan.md output format specification injected
// into the system prompt when the agent is asked to produce a plan document.
// It constrains the agent to use the task-block format required by §8.31.
//
// Constraint: ≤ 2500 chars.
const PlanTaskFormat = `
## PLAN.md Output Format Requirements

When writing tasks for the implementation plan (plan.md output document), use ONLY the following format. Do NOT use "Phase N" headings.

### Task N — {Short Task Name}

**Goal:** {One sentence, ≤ 120 chars, starting with a verb: what this task produces.}

**Files to create:**
- ` + "`path/to/file.go`" + ` — {description}
  - ` + "`func FunctionName(params) (ReturnType, error)`" + ` — exported function signature
  - ` + "`type TypeName struct { ... }`" + ` — key exported type
  - **Failure handling:** {how errors are returned/logged; which are fatal vs non-fatal}

**Validation:**
- ` + "`cd backend && go build ./...`" + `: zero build errors
- ` + "`cd backend && go vet ./...`" + `: zero vet issues
- ` + "`cd backend && go test ./internal/modules/X/...`" + `: all pass
- Manual smoke: {one behavioral assertion}

**Blocking Dependencies:** Task N, Task M

---

FORBIDDEN in Validation field: "per module test suite", "run tests", "validate implementation"
FORBIDDEN in Files field: "see phase deliverables", "TBD", "as needed"
REQUIRED in every task: at least one runnable shell command in Validation
REQUIRED for Go tasks: ` + "`go build ./...`" + ` in Validation
REQUIRED for frontend tasks: ` + "`pnpm check`" + ` in Validation
`

// InjectIfPlanOutput injects PlanTaskFormat into basePrompt when "plan" is
// present in outputDocs. Returns basePrompt unchanged when "plan" is absent.
//
// This implements Approach C (§8.31.3): the format spec is appended to the
// system prompt so the LLM sees it as a hard constraint, not a user request.
func InjectIfPlanOutput(outputDocs []string, basePrompt string) string {
	for _, doc := range outputDocs {
		if strings.EqualFold(strings.TrimSpace(doc), "plan") {
			return basePrompt + "\n\n" + strings.TrimLeft(PlanTaskFormat, "\n")
		}
	}
	return basePrompt
}
