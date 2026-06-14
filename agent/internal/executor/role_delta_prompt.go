package executor

// roleDeltaInstruction returns the per-role output contract. Agents read the
// full canonical state but emit only the keys they own — the backend merges
// the delta onto the running state (§8.5).
func roleDeltaInstruction(role string) string {
	switch role {
	case "build":
		return `ROLE OUTPUT (build): Return a JSON object with ONLY the keys you created or changed.
Include when applicable:
  - "idea" — refine the product idea if needed
  - "architecture" — layers, data_flows, tech_stack, decisions (your primary output)
  - "execution_plan" — phased plan steps with structured fields
  - "metrics" — include "confidence" when you can estimate it
Do NOT echo unchanged sections from the input state. Omit any key you did not modify.`

	case "review":
		return `ROLE OUTPUT (review): Return a JSON object with ONLY critique you added.
Include when applicable:
  - "risks" — new risks with severity and resolved=false
  - "open_questions" — gaps, ambiguities, missing requirements
  - "assumptions" — only if you challenge or add assumptions
Do NOT repeat architecture or execution_plan unless you are correcting a specific field.`

	case "refine":
		return `ROLE OUTPUT (refine): Return a JSON object with ONLY reconciled fields.
Include when applicable:
  - "execution_plan" — merged, de-duplicated, improved steps
  - "assumptions" — resolved or tightened assumptions
  - "architecture" — only when fixing contradictions from prior agents
  - "metrics" — updated "confidence" after synthesis
Omit keys you did not change.`

	case "devils_advocate":
		return `ROLE OUTPUT (devils_advocate): Return a JSON object with ONLY stress-test findings.
Include when applicable:
  - "risks" — edge cases, failure modes, consensus challenges
  - "open_questions" — uncomfortable questions the team must answer
  - "assumptions" — assumptions that may be wrong
Omit unchanged keys.`

	default:
		return `Return a JSON object containing ONLY the canonical-state keys you modified.
Read the full input state for context but do not echo unchanged fields.`
	}
}

const deltaOutputPreamble = `CRITICAL INSTRUCTION: You MUST respond with ONLY one valid JSON object (a delta).
Do NOT include explanation, commentary, markdown fences, or text outside the JSON.
Your entire response must start with { and end with }.
No prose before or after. No ` + "```" + `json fences. No trailing commas. Pure JSON only.
Return ONLY the keys your role owns — omit every unchanged field from the input state.
Before sending, verify the object parses as JSON.`

const continueTruncatedJSONPrompt = `Your previous JSON response was truncated before it was complete.
Continue EXACTLY where you stopped. Output ONLY the remaining JSON characters needed to finish the object.
Do NOT repeat content already sent. No prose. No markdown fences. No leading { unless the prior part ended mid-key.
When the prior part and this continuation are concatenated, the result must be one valid JSON object.`
