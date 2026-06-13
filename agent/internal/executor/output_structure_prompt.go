// Package executor — §8.23 output-structure prompt fragment.
//
// This text is appended to every SystemPrompt before the LLM is called. It
// describes the typed canonical-state schema agents may emit in their delta.
// Agents return role-scoped subsets only — omit unchanged keys.
package executor

const requiredOutputStructurePrompt = `

# Required Output Structure (canonical state schema)

You MUST return a JSON object (a delta) that conforms to the canonical brainstorm
state schema. Include only keys your role owns or changed — omit unchanged fields.
The following typed fields enable downstream document generation:

## architecture (object)

- ` + "`layers`" + ` (array of objects, optional):
  Each layer object has:
    - ` + "`name`" + ` (string, required) — e.g. "Backend API"
    - ` + "`responsibility`" + ` (string)
    - ` + "`technologies`" + ` (array of strings)
    - ` + "`dependencies`" + ` (array of strings, names of other layers)

- ` + "`data_flows`" + ` (array of objects, optional):
  Each flow object has:
    - ` + "`from`" + ` (string, required)
    - ` + "`to`" + ` (string, required)
    - ` + "`label`" + ` (string, optional, short verb phrase)

- ` + "`tech_stack`" + `, ` + "`decisions`" + `, ` + "`directory_layout`" + `,
  ` + "`config`" + ` (free-form, used as supplied).

## execution_plan (array of step objects)

Each step SHOULD include these structured fields when applicable:

- ` + "`title`" + ` (string, required, short phase name)
- ` + "`description`" + ` (string, one-paragraph summary)
- ` + "`objective`" + ` (string, one-sentence purpose)
- ` + "`blocking_dependencies`" + ` (array of strings, names of phases that
  must complete first)
- ` + "`scope`" + ` (string, what is in / out of this phase)
- ` + "`deliverables`" + ` (array of strings, concrete output artifacts)
- ` + "`function_contracts`" + ` (array of strings, e.g. signatures
  "ParseInput(input string) (Token, error)")
- ` + "`failure_handling`" + ` (string, retry / abort policy)
- ` + "`exit_criteria`" + ` (array of strings, testable completion checks)

## metrics (object)

- ` + "`confidence`" + ` (number, 0.0–1.0, required)
- ` + "`test_coverage_target`" + ` (number, 0.0–1.0, optional)
- ` + "`latency_budget_ms`" + ` (integer, optional)

## Style rules

- Return plain JSON only — no Markdown fences, no commentary, no prose outside the object.
- DO NOT repeat the project idea text inside any field other than ` + "`idea`" + `.
- DO NOT pad lists with placeholder filler entries; only include real items.
- Keep ` + "`risks`" + ` array entries concrete, with severity and resolved flag.
- Verify the final answer parses as valid JSON before you send it.
`
