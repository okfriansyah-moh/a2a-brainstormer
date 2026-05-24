// Package markdown — architecture document generator.
// GenerateArchitecture follows the §8.20 twelve-section skeleton and wraps
// the body in enforceMinLines so the output is always ≥ 1000 lines.
package markdown

import (
	"fmt"
	"slices"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateArchitecture renders the architecture.md document from s.
// It follows the §8.20 section skeleton (12 sections) and enforces a minimum
// of 1000 lines via the padArchitecture padder.
func GenerateArchitecture(s state.CanonicalState) (string, error) {
	var b strings.Builder

	// ── Title ────────────────────────────────────────────────────────────────
	title := "Architecture Document"
	if text, ok := s.Idea["text"]; ok {
		title = fmt.Sprintf("%v — Architecture", text)
	}
	b.WriteString(fmt.Sprintf("# %s\n\n", title))

	// ── § 1. Overview ────────────────────────────────────────────────────────
	b.WriteString("## 1. Overview\n\n")
	if len(s.Idea) > 0 {
		b.WriteString("### Project Idea\n\n")
		writeMap(&b, s.Idea)
	}
	if len(s.Architecture) > 0 {
		b.WriteString("### Architecture Summary\n\n")
		b.WriteString("The system is composed of the following architectural layers:\n\n")
		writeMap(&b, s.Architecture)
	} else {
		b.WriteString("_Architecture details not yet defined._\n\n")
	}
	b.WriteString(fmt.Sprintf("> Confidence: **%.4f** — Iteration: **%d**\n\n",
		s.Metrics.Confidence, s.Meta.Iteration))

	// ── § 2. System Components ───────────────────────────────────────────────
	b.WriteString("## 2. System Components\n\n")
	if len(s.Architecture) > 0 {
		keys := make([]string, 0, len(s.Architecture))
		for k := range s.Architecture {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("### %s\n\n", k))
			b.WriteString(fmt.Sprintf("**Technology / Description:** %v\n\n", s.Architecture[k]))
			b.WriteString("**Key Responsibilities:**\n\n")
			b.WriteString(fmt.Sprintf("- Own the `%s` domain boundary.\n", k))
			b.WriteString("- Expose a typed service interface to HTTP handlers.\n")
			b.WriteString("- Persist state exclusively through its own repository.\n")
			b.WriteString("- Emit structured log entries for every state transition.\n\n")
			b.WriteString("**Communication Pattern:** synchronous HTTP (internal), A2A protocol (agent dispatch).\n\n")
		}
	} else {
		b.WriteString("_No components defined yet. Agents will populate this section during iteration._\n\n")
	}

	// ── § 3. Data Flow ───────────────────────────────────────────────────────
	b.WriteString("## 3. Data Flow\n\n")
	b.WriteString("### Component Interaction Diagram\n\n")
	b.WriteString(renderASCIIComponents(s))
	b.WriteString("### Request Lifecycle\n\n")
	b.WriteString("1. Client sends HTTP request to the backend API.\n")
	b.WriteString("2. Router dispatches to the appropriate module handler.\n")
	b.WriteString("3. Handler validates input and delegates to the service layer.\n")
	b.WriteString("4. Service orchestrates business logic and calls repository/platform layers.\n")
	b.WriteString("5. Repository executes parameterised SQL via pgx.\n")
	b.WriteString("6. Response is serialised to JSON and returned to the client.\n\n")
	b.WriteString("### Agent Dispatch Flow\n\n")
	b.WriteString("1. Iteration engine loads session + agents from DB.\n")
	b.WriteString("2. For each agent: assemble prompt (system + skills + canonical state).\n")
	b.WriteString("3. Send `SendMessageRequest` via `a2aclient.NewFromCard`.\n")
	b.WriteString("4. Agent binary receives the request, calls LLMProvider, returns updated state.\n")
	b.WriteString("5. Iteration engine merges agent outputs into the canonical state.\n")
	b.WriteString("6. Check convergence; persist state; repeat if not converged.\n\n")

	// ── § 4. Tech Stack ──────────────────────────────────────────────────────
	b.WriteString("## 4. Tech Stack\n\n")
	b.WriteString(renderTechStack(s))

	// ── § 5. Module Boundaries ───────────────────────────────────────────────
	b.WriteString("## 5. Module Boundaries\n\n")
	b.WriteString("The backend is a **modular monolith** with strict import rules:\n\n")
	b.WriteString("| Rule | Description |\n")
	b.WriteString("|------|-------------|\n")
	b.WriteString("| No cross-module internal imports | `session` must not import `agent/repository` |\n")
	b.WriteString("| Shared types in `internal/shared/` | Used by multiple modules; no owner conflict |\n")
	b.WriteString("| Platform in `internal/platform/` | Any module may import; platform imports nothing from modules |\n")
	b.WriteString("| LLM calls via LLMProvider only | No direct SDK in `internal/modules/` |\n")
	b.WriteString("| DB access via own repository only | No module queries another module's tables |\n\n")
	b.WriteString("### Module Directory Structure\n\n")
	b.WriteString(renderDirectoryTree(s))

	// ── § 6. Key Architecture Decisions ─────────────────────────────────────
	b.WriteString("## 6. Key Architecture Decisions\n\n")
	b.WriteString(renderDecisionsTable(s))

	// ── § 7. Data Model ──────────────────────────────────────────────────────
	b.WriteString("## 7. Data Model\n\n")
	b.WriteString("### Core Entities\n\n")
	b.WriteString(renderTable(
		[]string{"Entity", "Table", "Key Fields", "Notes"},
		[][]string{
			{"Session", "sessions", "id, status, output_docs, canonical_state", "Central orchestration unit"},
			{"Agent", "agents", "id, name, system_prompt, llm_config", "LLM-backed design participant"},
			{"Skill", "skills", "id, name, prompt", "Prompt fragment injected at dispatch time"},
			{"SessionAgent", "session_agents", "session_id, agent_id, position, role", "Ordered pipeline slot"},
			{"AgentSkill", "agent_skills", "agent_id, skill_id", "Default skill set for an agent"},
		},
	))
	b.WriteString("### State Storage\n\n")
	b.WriteString("- `sessions.canonical_state` is a `JSONB` column holding the full `CanonicalState`.\n")
	b.WriteString("- Updated atomically after each full pipeline pass.\n")
	b.WriteString("- Version field enables optimistic concurrency.\n\n")

	// ── § 8. API Surface ─────────────────────────────────────────────────────
	b.WriteString("## 8. API Surface\n\n")
	b.WriteString("All endpoints are REST/JSON over HTTP. No authentication in v1.\n\n")
	b.WriteString(renderTable(
		[]string{"Method", "Path", "Request Body", "Response"},
		[][]string{
			{"POST", "/sessions", "CreateSessionRequest", "201 Session"},
			{"GET", "/sessions", "—", "200 []SessionListItem"},
			{"GET", "/sessions/{id}", "—", "200 Session"},
			{"PATCH", "/sessions/{id}/output-docs", "UpdateOutputDocsRequest", "200 Session"},
			{"POST", "/sessions/{id}/iterate", "—", "200 CanonicalState"},
			{"POST", "/sessions/{id}/finalize", "FinalizeInput?", "200 FinalizeResponse"},
			{"GET", "/agents", "—", "200 []Agent"},
			{"POST", "/agents", "CreateAgentRequest", "201 Agent"},
			{"GET", "/agents/{id}", "—", "200 Agent"},
			{"PUT", "/agents/{id}", "UpdateAgentRequest", "200 Agent"},
			{"DELETE", "/agents/{id}", "—", "204"},
		},
	))

	// ── § 9. Failure Modes ───────────────────────────────────────────────────
	b.WriteString("## 9. Failure Modes\n\n")
	b.WriteString(renderRisksTable(s))
	b.WriteString("### Error Response Shape\n\n")
	b.WriteString("All error responses use a consistent JSON envelope:\n\n")
	b.WriteString("```json\n")
	b.WriteString("{\n")
	b.WriteString("  \"error\": \"descriptive message\",\n")
	b.WriteString("  \"code\": \"ERROR_CODE\"\n")
	b.WriteString("}\n")
	b.WriteString("```\n\n")

	// ── § 10. Observability ──────────────────────────────────────────────────
	b.WriteString("## 10. Observability\n\n")
	b.WriteString("### Logging\n\n")
	b.WriteString("- Structured logging via `log/slog` (stdlib).\n")
	b.WriteString("- Every request carries a `request_id` propagated through context.\n")
	b.WriteString("- Agent dispatch events logged at `INFO` with duration, agent ID, iteration.\n")
	b.WriteString("- Errors logged at `ERROR` with full wrapped error chain.\n\n")
	b.WriteString("### Metrics\n\n")
	b.WriteString("- `iteration_duration_seconds` — histogram per session.\n")
	b.WriteString("- `agent_dispatch_duration_seconds` — histogram per agent.\n")
	b.WriteString("- `convergence_confidence` — gauge per session.\n")
	b.WriteString("- `http_request_duration_seconds` — histogram per endpoint.\n\n")
	b.WriteString("### Health Check\n\n")
	b.WriteString("- `GET /healthz` — returns `200 {\"status\": \"ok\"}` when DB is reachable.\n\n")

	// ── § 11. Security Model ─────────────────────────────────────────────────
	b.WriteString("## 11. Security Model\n\n")
	b.WriteString("| Control | Implementation |\n")
	b.WriteString("|---------|----------------|\n")
	b.WriteString("| Secret storage | Environment variables only; never in source or DB |\n")
	b.WriteString("| SQL injection | Parameterised queries via pgx; no interpolation |\n")
	b.WriteString("| Input validation | Every handler validates shape, type, and bounds |\n")
	b.WriteString("| Error disclosure | Internal errors wrapped; raw messages never forwarded |\n")
	b.WriteString("| CORS | Configured origins allowlist; wildcard forbidden in production |\n")
	b.WriteString("| Dependency audit | `go mod tidy` + `govulncheck` in CI pipeline |\n\n")

	// ── § 12. Open Questions ─────────────────────────────────────────────────
	b.WriteString("## 12. Open Questions\n\n")
	if len(s.OpenQuestions) > 0 {
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_No open questions at this time._\n\n")
	}

	b.WriteString(fmt.Sprintf("---\n_Generated at iteration %d. Confidence: %.4f_\n", s.Meta.Iteration, s.Metrics.Confidence))

	body := b.String()
	return enforceMinLines(body, s, padArchitecture), nil
}
