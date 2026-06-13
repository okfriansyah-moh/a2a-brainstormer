// Package prompts — Approach C prompt injection for readme output format (§8.32.5).
//
// This package must NOT import any backend/internal/ packages. It is a
// self-contained constant + helper used by the agent executor to inject the
// readme format when the payload's OutputDocs includes "readme".
package prompts

// ReadmeFormat is the canonical readme.md output format specification injected
// into the system prompt when the agent is asked to produce a readme document.
// It constrains the agent to use the README section format required by §8.32.
//
// Constraint: ≤ 3000 chars.
const ReadmeFormat = `=== README OUTPUT FORMAT ===

When generating content for the readme.md document, write every section in README format (not roadmap or plan format):

REQUIRED SECTIONS (in this order):
1. Golden rule — one sentence stating the single most important system invariant. Format: "**Golden rule:** {sentence}"
2. What it is NOT — 3-5 bullet points clarifying scope boundaries. Start each with "Not a ..." or "Not designed to ...".
3. When to use — numbered scenarios (3-6), each with: title, 1-2 sentence description, fenced code example showing a real command or config snippet.
4. Installation / Prerequisites — a table with | Tool | Version | Required | columns followed by setup steps.
5. Quick Start — a fenced bash block with 2-5 REAL runnable commands for THIS project's tech stack. NEVER write: git clone <repository-url>, make dev (unless Makefile is confirmed), placeholder commands.
6. Command Reference — a markdown table with | Command | Description | columns. Include build, test, run, and the 3-5 most important project-specific commands.
7. Architecture — first an ASCII diagram (box-drawing chars), then a mermaid flowchart. Both must show the actual components described in the architecture state.
8. Contributing — 2-3 sentences: what documentation to read first, how to run tests before submitting.

FORBIDDEN:
- "Phase N —" bullets anywhere in README output
- "## Known Risks" as a top-level section
- "## Roadmap" section with milestone bullets
- Placeholder text: <repository-url>, <project>, TODO, TBD
- Generic filler: "Add your description here", "Coming soon"

The README must be specific to THIS project. Use the actual project name, tech stack, and module structure from the conversation context.
=== END README FORMAT ===`

// InjectIfReadmeOutput appends ReadmeFormat to base when "readme" is in
// outputDocs. Returns base unchanged when "readme" is absent.
//
// This implements Approach C (§8.32.5): the format spec is appended to the
// system prompt so the LLM sees it as a hard constraint, not a user request.
func InjectIfReadmeOutput(base string, outputDocs []string) string {
	for _, doc := range outputDocs {
		if doc == "readme" {
			return base + "\n\n" + ReadmeFormat
		}
	}
	return base
}
