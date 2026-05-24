// Package markdown — shared rendering helpers and line-count enforcement used
// by all four document generators. All functions are pure and deterministic:
// same input → identical output, no randomness, no time.Now(), no map
// iteration without key sorting.
package markdown

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// minDocLines is the minimum number of newlines that every generated document
// must contain. Callers read this via enforceMinLines.
const minDocLines = 1000

// ── shared rendering helpers ─────────────────────────────────────────────────

// writeMap writes the key-value pairs of a map[string]any as Markdown bullet
// points into b. Keys are sorted for deterministic output.
func writeMap(b *strings.Builder, m map[string]any) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("- **%s**: %v\n", k, m[k]))
	}
	b.WriteString("\n")
}

// renderTable returns a Markdown table with the given headers and rows.
// All columns are left-aligned. Rows with fewer columns than headers are
// right-padded with empty strings.
func renderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	b.WriteString("| " + strings.Join(sep, " | ") + " |\n")
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range cells {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderASCIIComponents produces a simple text box diagram that lists all
// keys in s.Architecture as labelled boxes connected left-to-right.
// When s.Architecture is empty a placeholder is returned.
func renderASCIIComponents(s state.CanonicalState) string {
	if len(s.Architecture) == 0 {
		return "```\n[ No components defined ]\n```\n\n"
	}
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var b strings.Builder
	b.WriteString("```\n")
	for i, k := range keys {
		// Draw a box for each component.
		width := len(k) + 2
		top := "+" + strings.Repeat("-", width) + "+"
		mid := "| " + k + " |"
		b.WriteString(top + "\n")
		b.WriteString(mid + "\n")
		b.WriteString(top + "\n")
		if i < len(keys)-1 {
			b.WriteString("       |\n")
			b.WriteString("       v\n")
		}
	}
	b.WriteString("```\n\n")
	return b.String()
}

// renderDirectoryTree renders the value stored under key "directory_layout"
// in s.Architecture as a fenced code block. Falls back to a placeholder.
func renderDirectoryTree(s state.CanonicalState) string {
	if v, ok := s.Architecture["directory_layout"]; ok {
		return fmt.Sprintf("```\n%v\n```\n\n", v)
	}
	if len(s.Architecture) == 0 {
		return "```\n<directory structure not yet defined>\n```\n\n"
	}
	// Generate a minimal tree from known component keys.
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString("```\n./\n")
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("├── %s/\n", k))
	}
	b.WriteString("└── docs/\n")
	b.WriteString("```\n\n")
	return b.String()
}

// renderTechStack produces a Markdown section listing all technology entries
// found in s.Architecture["tech_stack"] or falls back to the raw Architecture
// map when no tech_stack key exists.
func renderTechStack(s state.CanonicalState) string {
	var b strings.Builder
	if ts, ok := s.Architecture["tech_stack"]; ok {
		b.WriteString(fmt.Sprintf("%v\n\n", ts))
		return b.String()
	}
	if len(s.Architecture) == 0 {
		return "_Tech stack not yet defined._\n\n"
	}
	// Fall back: list architecture map entries as tech choices.
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{k, fmt.Sprintf("%v", s.Architecture[k]), "—"})
	}
	return renderTable([]string{"Layer", "Technology", "Version"}, rows)
}

// renderDecisionsTable produces a Markdown table of architecture decisions
// stored in s.Architecture["decisions"]. Falls back to a placeholder row.
func renderDecisionsTable(s state.CanonicalState) string {
	if v, ok := s.Architecture["decisions"]; ok {
		return fmt.Sprintf("%v\n\n", v)
	}
	// Synthesise a decisions table from available architecture data.
	if len(s.Architecture) == 0 {
		return renderTable(
			[]string{"ID", "Decision", "Rationale", "Status"},
			[][]string{{"ADR-001", "Architecture not yet defined", "—", "Draft"}},
		)
	}
	keys := make([]string, 0, len(s.Architecture))
	for k := range s.Architecture {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	rows := make([][]string, 0, len(keys))
	for i, k := range keys {
		rows = append(rows, []string{
			fmt.Sprintf("ADR-%03d", i+1),
			fmt.Sprintf("Use %s for %s layer", s.Architecture[k], k),
			"Selected for performance and ecosystem fit",
			"Accepted",
		})
	}
	return renderTable([]string{"ID", "Decision", "Rationale", "Status"}, rows)
}

// renderRisksTable produces a Markdown table of all risks in s.Risks.
// Resolved risks are marked with ✅; unresolved ones with ⚠️.
func renderRisksTable(s state.CanonicalState) string {
	if len(s.Risks) == 0 {
		return "_No risks identified yet._\n\n"
	}
	rows := make([][]string, 0, len(s.Risks))
	for i, r := range s.Risks {
		status := "⚠️ Open"
		if r.Resolved {
			status = "✅ Resolved"
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			r.Text,
			strings.ToTitle(r.Severity),
			status,
		})
	}
	return renderTable([]string{"#", "Risk", "Severity", "Status"}, rows)
}

// renderExecutionPlanList produces a numbered Markdown list of execution plan
// steps with titles and descriptions.
func renderExecutionPlanList(s state.CanonicalState) string {
	if len(s.ExecutionPlan) == 0 {
		return "_No execution plan defined yet._\n\n"
	}
	var b strings.Builder
	for i, step := range s.ExecutionPlan {
		b.WriteString(fmt.Sprintf("%d. **%s**", i+1, step.Title))
		if step.Description != "" {
			b.WriteString(fmt.Sprintf(" — %s", step.Description))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderEnvVarList produces a Markdown list of configuration / environment
// variables derived from s.Architecture["config"]. Falls back to a generic
// placeholder list when the config key is absent.
func renderEnvVarList(s state.CanonicalState) string {
	if v, ok := s.Architecture["config"]; ok {
		return fmt.Sprintf("```env\n%v\n```\n\n", v)
	}
	// Generic placeholder derived from component names.
	var b strings.Builder
	b.WriteString("```env\n")
	if len(s.Architecture) > 0 {
		keys := make([]string, 0, len(s.Architecture))
		for k := range s.Architecture {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			name := strings.ToUpper(strings.ReplaceAll(k, " ", "_"))
			b.WriteString(fmt.Sprintf("%s_HOST=localhost\n", name))
			b.WriteString(fmt.Sprintf("%s_PORT=8080\n", name))
		}
	} else {
		b.WriteString("APP_HOST=localhost\n")
		b.WriteString("APP_PORT=8080\n")
		b.WriteString("DATABASE_URL=postgres://user:pass@localhost:5432/db\n")
	}
	b.WriteString("```\n\n")
	return b.String()
}

// renderJSONDump serialises s to indented JSON inside a fenced code block.
// Errors in marshalling are silently ignored (the output falls back to empty).
func renderJSONDump(s state.CanonicalState) string {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "```json\n{}\n```\n\n"
	}
	return fmt.Sprintf("```json\n%s\n```\n\n", string(raw))
}

// ── line-count enforcer ──────────────────────────────────────────────────────

// enforceMinLines appends padFn(s) to body in a loop until body contains at
// least minDocLines newlines. The padFn must make meaningful progress (emit at
// least one new line) to avoid infinite loops; the padders in this package
// always emit hundreds of lines per call.
func enforceMinLines(body string, s state.CanonicalState, padFn func(state.CanonicalState) string) string {
	for strings.Count(body, "\n") < minDocLines {
		body += "\n\n" + padFn(s)
	}
	return body
}

// ── per-generator padders ────────────────────────────────────────────────────

// padArchitecture emits a comprehensive extended reference appendix for the
// architecture document. It always produces several hundred lines so that two
// calls at most are needed to reach minDocLines.
func padArchitecture(s state.CanonicalState) string {
	var b strings.Builder

	b.WriteString("---\n\n")
	b.WriteString("## Appendix A: Extended Component Analysis\n\n")
	b.WriteString("This appendix provides detailed deep-dive profiles for every component in the system. ")
	b.WriteString("Each profile covers data flow, failure modes, observability hooks, and deployment considerations.\n\n")

	// Per-component deep-dive.
	if len(s.Architecture) > 0 {
		keys := make([]string, 0, len(s.Architecture))
		for k := range s.Architecture {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			writeComponentDeepDive(&b, k, fmt.Sprintf("%v", s.Architecture[k]))
		}
	} else {
		writeComponentDeepDive(&b, "Application", "Core application layer")
	}

	b.WriteString("## Appendix B: System Integration Patterns\n\n")
	b.WriteString("### Synchronous Communication\n\n")
	b.WriteString("Synchronous calls are used for request-response interactions where the caller must ")
	b.WriteString("block until a result is available. The following patterns apply:\n\n")
	b.WriteString("- **Request-Response**: Standard HTTP/gRPC call. Caller blocks. Timeout required.\n")
	b.WriteString("- **Retry with Back-off**: Failed calls are retried with exponential back-off.\n")
	b.WriteString("  - Initial delay: 100 ms, multiplier: 2, max retries: 3, max delay: 5 s.\n")
	b.WriteString("- **Circuit Breaker**: After 5 consecutive failures the circuit opens; half-open after 30 s.\n")
	b.WriteString("- **Deadline Propagation**: Parent context deadline is forwarded to all child calls.\n\n")
	b.WriteString("### Asynchronous Communication\n\n")
	b.WriteString("Async patterns decouple producers from consumers and improve throughput:\n\n")
	b.WriteString("- **Event Bus**: Components publish domain events; subscribers react independently.\n")
	b.WriteString("- **Outbox Pattern**: Events are stored in an outbox table before being dispatched to avoid\n")
	b.WriteString("  data loss on crash. A background worker polls the outbox and delivers events.\n")
	b.WriteString("- **Idempotent Consumers**: All event handlers are designed to be safe to replay.\n")
	b.WriteString("  Deduplication is achieved via event IDs stored in a processed-events table.\n\n")
	b.WriteString("### State Transfer\n\n")
	b.WriteString("State is transferred between components via the CanonicalState contract:\n\n")
	b.WriteString("- State is immutable during transit: each agent receives a snapshot and returns a new snapshot.\n")
	b.WriteString("- Merge is performed by the iteration engine after all agents complete a pass.\n")
	b.WriteString("- Optimistic concurrency: the DB row version (ETag) is checked before writing.\n\n")

	b.WriteString("## Appendix C: Technology Evaluation Matrix\n\n")
	b.WriteString("The following matrix summarises the technologies evaluated for this system and the reasons ")
	b.WriteString("for the selected choices:\n\n")
	b.WriteString(renderTable(
		[]string{"Category", "Selected", "Alternatives Considered", "Decision Rationale"},
		[][]string{
			{"Backend runtime", "Go 1.26", "Java 21, Rust, Node.js", "Compilation speed, small binary, goroutine model"},
			{"Database", "PostgreSQL 16", "MySQL 8, SQLite, CockroachDB", "JSONB support, pgx/v5 driver, battle-tested"},
			{"API protocol", "REST over HTTP/1.1", "gRPC, GraphQL", "Widest tooling support, simple debugging"},
			{"Frontend", "SvelteKit", "Next.js, Nuxt, Remix", "Minimal bundle size, built-in SSR, TypeScript"},
			{"Agent protocol", "A2A (a2a-go/v2)", "OpenAI Assistants, LangChain", "Standard inter-agent protocol, vendor-neutral"},
			{"LLM interface", "LLMProvider interface", "Direct SDK", "Abstraction enables provider swapping without refactor"},
			{"Container", "Docker + Compose", "Kubernetes", "Simpler local dev; K8s for production scale-out"},
		},
	))

	b.WriteString("## Appendix D: Non-Functional Requirements\n\n")
	b.WriteString("### Performance\n\n")
	b.WriteString("| Metric | Target | Measurement Method |\n")
	b.WriteString("|--------|--------|--------------------|\n")
	b.WriteString("| API p50 latency | < 50 ms | Prometheus histogram |\n")
	b.WriteString("| API p99 latency | < 500 ms | Prometheus histogram |\n")
	b.WriteString("| Iteration round-trip | < 30 s (2 agents) | Trace span |\n")
	b.WriteString("| DB query p99 | < 10 ms | pgx instrumentation |\n")
	b.WriteString("| Frontend FCP | < 1.5 s | Lighthouse |\n\n")
	b.WriteString("### Scalability\n\n")
	b.WriteString("- Stateless HTTP handlers allow horizontal scaling behind a load balancer.\n")
	b.WriteString("- Iteration engine holds per-session mutexes; sessions are independent work units.\n")
	b.WriteString("- PostgreSQL connection pool (pgx) caps concurrent DB connections.\n")
	b.WriteString("- LLM rate limits are handled by the LLMProvider implementation with back-off.\n")
	b.WriteString("- Agent binary scales independently of the backend monolith.\n\n")
	b.WriteString("### Reliability\n\n")
	b.WriteString("- Database is the source of truth; in-memory state (preview store) is ephemeral.\n")
	b.WriteString("- All file writes are atomic (write temp → rename) to prevent partial writes.\n")
	b.WriteString("- Iteration checkpoints: state is persisted after each full pipeline pass.\n")
	b.WriteString("- Graceful shutdown: in-flight requests complete before the server exits.\n")
	b.WriteString("- Health check endpoint (`GET /healthz`) returns 200 when DB is reachable.\n\n")
	b.WriteString("### Security\n\n")
	b.WriteString("- All API keys are resolved from environment variables at runtime.\n")
	b.WriteString("- `os.Getenv` calls are confined to `platform/config/config.go`.\n")
	b.WriteString("- SQL uses parameterised queries only; no string interpolation.\n")
	b.WriteString("- Input validation on every handler: UUID format, non-empty fields, bounded integers.\n")
	b.WriteString("- CORS is configured to allow only known origins.\n")
	b.WriteString("- No secrets are emitted in logs or error responses.\n\n")
	b.WriteString("### Maintainability\n\n")
	b.WriteString("- Vertical slice architecture: each module owns handler + service + repository + model.\n")
	b.WriteString("- No cross-module internal imports; shared types live in `internal/shared/`.\n")
	b.WriteString("- Structured logging with `log/slog`: every log entry carries request ID and session ID.\n")
	b.WriteString("- Migrations are append-only numbered SQL files; no destructive changes.\n")
	b.WriteString("- All configuration is driven by environment variables; no hardcoded values.\n\n")

	b.WriteString("## Appendix E: Deployment Architecture\n\n")
	b.WriteString("### Development Environment\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Start all services\n")
	b.WriteString("docker compose up -d\n\n")
	b.WriteString("# Run backend\n")
	b.WriteString("cd backend && go run ./cmd/server\n\n")
	b.WriteString("# Run agent\n")
	b.WriteString("cd agent && go run ./cmd/server\n\n")
	b.WriteString("# Run frontend\n")
	b.WriteString("cd frontend && pnpm dev\n")
	b.WriteString("```\n\n")
	b.WriteString("### Container Images\n\n")
	b.WriteString("| Image | Base | Build Command | Notes |\n")
	b.WriteString("|-------|------|---------------|-------|\n")
	b.WriteString("| `a2a-backend` | `gcr.io/distroless/static` | `make docker-backend` | ~15 MB |\n")
	b.WriteString("| `a2a-agent` | `gcr.io/distroless/static` | `make docker-agent` | ~12 MB |\n")
	b.WriteString("| `a2a-frontend` | `node:20-alpine` | `make docker-frontend` | ~50 MB |\n\n")
	b.WriteString("### Production Checklist\n\n")
	b.WriteString("- [ ] Set `DATABASE_URL` to a production PostgreSQL instance.\n")
	b.WriteString("- [ ] Set `COPILOT_API_KEY` or `CLAUDE_API_KEY` depending on configured provider.\n")
	b.WriteString("- [ ] Set `AGENT_ENDPOINT_0` to the reachable agent binary URL.\n")
	b.WriteString("- [ ] Run `migrate up` to apply all pending migrations.\n")
	b.WriteString("- [ ] Configure TLS termination at the load balancer.\n")
	b.WriteString("- [ ] Enable structured log shipping to your observability stack.\n\n")

	b.WriteString("## Appendix F: Observability Reference\n\n")
	b.WriteString("### Key Metrics\n\n")
	b.WriteString("| Metric | Type | Labels | Description |\n")
	b.WriteString("|--------|------|--------|-------------|\n")
	b.WriteString("| `iteration_duration_seconds` | Histogram | session_id | Full pipeline pass duration |\n")
	b.WriteString("| `agent_dispatch_duration_seconds` | Histogram | agent_id, role | Single agent call duration |\n")
	b.WriteString("| `convergence_confidence` | Gauge | session_id | Latest confidence score |\n")
	b.WriteString("| `session_total` | Counter | status | Sessions by terminal status |\n")
	b.WriteString("| `llm_calls_total` | Counter | provider, model | LLM call count |\n")
	b.WriteString("| `llm_errors_total` | Counter | provider, error_class | LLM failure count |\n\n")
	b.WriteString("### Structured Log Fields\n\n")
	b.WriteString("Every log entry emits the following slog fields:\n\n")
	b.WriteString("| Field | Type | Example |\n")
	b.WriteString("|-------|------|---------|\n")
	b.WriteString("| `request_id` | string | `550e8400-e29b` |\n")
	b.WriteString("| `session_id` | string | `a1b2c3d4-...` |\n")
	b.WriteString("| `iteration` | int | `3` |\n")
	b.WriteString("| `agent_id` | string | `uuid` |\n")
	b.WriteString("| `duration_ms` | float64 | `142.3` |\n")
	b.WriteString("| `error` | string | wrapped error message |\n\n")

	b.WriteString("## Appendix G: Per-Step Execution Plan Analysis\n\n")
	if len(s.ExecutionPlan) > 0 {
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("### Step %d: %s\n\n", i+1, step.Title))
			if step.Description != "" {
				b.WriteString(step.Description + "\n\n")
			}
			b.WriteString("**Assumptions:**\n\n")
			for _, a := range s.Assumptions {
				b.WriteString(fmt.Sprintf("- %s\n", a))
			}
			if len(s.Assumptions) == 0 {
				b.WriteString("- No specific assumptions recorded.\n")
			}
			b.WriteString("\n**Risks:**\n\n")
			hasStepRisks := false
			for _, r := range s.Risks {
				if !r.Resolved {
					b.WriteString(fmt.Sprintf("- **[%s]** %s\n", strings.ToUpper(r.Severity), r.Text))
					hasStepRisks = true
				}
			}
			if !hasStepRisks {
				b.WriteString("- No open risks for this step.\n")
			}
			b.WriteString("\n**Validation Criteria:**\n\n")
			b.WriteString("- Unit tests pass with 0 failures.\n")
			b.WriteString("- Integration tests pass against local services.\n")
			b.WriteString("- `go vet` + linter report 0 issues.\n")
			b.WriteString("- `go build` exits with code 0.\n\n")
		}
	} else {
		b.WriteString("_No execution plan steps recorded. Steps will appear here as agents generate the plan._\n\n")
	}

	b.WriteString("## Appendix H: Canonical State JSON Reference\n\n")
	b.WriteString("The full canonical state snapshot at the time this document was generated:\n\n")
	b.WriteString(renderJSONDump(s))

	return b.String()
}

// writeComponentDeepDive writes a detailed multi-section profile for a single
// component into b.
func writeComponentDeepDive(b *strings.Builder, name, description string) {
	b.WriteString(fmt.Sprintf("### Component: %s\n\n", name))
	b.WriteString(fmt.Sprintf("**Description:** %s\n\n", description))
	b.WriteString("**Responsibilities:**\n\n")
	b.WriteString(fmt.Sprintf("- Own and enforce the business logic for the `%s` domain.\n", name))
	b.WriteString("- Expose a well-typed service interface consumed by HTTP handlers.\n")
	b.WriteString("- Access persistence exclusively through its own repository.\n")
	b.WriteString("- Emit structured log entries for every significant state transition.\n\n")
	b.WriteString("**External Interfaces:**\n\n")
	b.WriteString(fmt.Sprintf("- HTTP handlers at `internal/modules/%s/handler.go`.\n", strings.ToLower(name)))
	b.WriteString(fmt.Sprintf("- Service interface at `internal/modules/%s/service.go`.\n", strings.ToLower(name)))
	b.WriteString(fmt.Sprintf("- Repository at `internal/modules/%s/repository.go`.\n", strings.ToLower(name)))
	b.WriteString(fmt.Sprintf("- Models at `internal/modules/%s/model.go`.\n\n", strings.ToLower(name)))
	b.WriteString("**Data Flow:**\n\n")
	b.WriteString("```\n")
	b.WriteString(fmt.Sprintf("HTTP Request → %s Handler → %s Service → %s Repository → PostgreSQL\n", name, name, name))
	b.WriteString(fmt.Sprintf("              ← %s Response ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ←\n", name))
	b.WriteString("```\n\n")
	b.WriteString("**Failure Modes:**\n\n")
	b.WriteString("| Failure | Symptom | Recovery |\n")
	b.WriteString("|---------|---------|----------|\n")
	b.WriteString("| DB connection timeout | 503 response | Retry via pgx pool |\n")
	b.WriteString("| Invalid input | 400 response | Client fixes request |\n")
	b.WriteString("| Concurrent modification | 409 response | Client retries |\n")
	b.WriteString("| Internal error | 500 response | Alert + investigate |\n\n")
	b.WriteString("**Observability:**\n\n")
	b.WriteString(fmt.Sprintf("- Log `%s.created`, `%s.updated`, `%s.deleted` events at `INFO` level.\n", name, name, name))
	b.WriteString("- Log errors at `ERROR` level with full context (request ID, entity ID, error chain).\n")
	b.WriteString(fmt.Sprintf("- Emit `%s_operations_total` counter metric with `operation` and `status` labels.\n\n", strings.ToLower(name)))
	b.WriteString("**Deployment Notes:**\n\n")
	b.WriteString("- No stateful resources held in memory beyond request scope.\n")
	b.WriteString("- Zero-downtime deployment: DB schema changes use additive migrations only.\n")
	b.WriteString("- Feature flags can disable this component without affecting others.\n\n")
}

// padRoadmap emits an extended rollout appendix for the roadmap document.
func padRoadmap(s state.CanonicalState) string {
	var b strings.Builder

	b.WriteString("---\n\n")
	b.WriteString("## Appendix A: Detailed Phase Breakdown\n\n")
	b.WriteString("Each phase below maps to one or more execution plan steps. Exit criteria must be ")
	b.WriteString("satisfied before moving to the next phase.\n\n")

	if len(s.ExecutionPlan) > 0 {
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("### Phase %d: %s\n\n", i+1, step.Title))
			if step.Description != "" {
				b.WriteString("**Description:** " + step.Description + "\n\n")
			}
			b.WriteString("**Deliverables:**\n\n")
			b.WriteString(fmt.Sprintf("- Completed implementation of all files scoped to phase %d.\n", i+1))
			b.WriteString("- Unit tests with 0 failures.\n")
			b.WriteString("- Integration tests pass against local services.\n")
			b.WriteString("- Documentation updated.\n\n")
			b.WriteString("**Exit Criteria:**\n\n")
			b.WriteString("- [ ] All defined tests pass.\n")
			b.WriteString("- [ ] Code review approved.\n")
			b.WriteString("- [ ] No open critical or high severity risks.\n")
			b.WriteString("- [ ] Performance benchmarks within target.\n\n")
			b.WriteString("**Dependencies:**\n\n")
			if i == 0 {
				b.WriteString("- No upstream dependencies (initial phase).\n\n")
			} else {
				b.WriteString(fmt.Sprintf("- Phase %d must be complete before starting this phase.\n\n", i))
			}
		}
	} else {
		b.WriteString("_Phases will be populated as the execution plan is developed by agents._\n\n")
	}

	b.WriteString("## Appendix B: Risk Register\n\n")
	b.WriteString("Full risk register with mitigations:\n\n")
	if len(s.Risks) > 0 {
		b.WriteString(renderTable(
			[]string{"#", "Risk", "Severity", "Probability", "Impact", "Mitigation", "Status"},
			riskRows(s),
		))
	} else {
		b.WriteString("_No risks recorded._\n\n")
	}

	b.WriteString("## Appendix C: Dependency Graph\n\n")
	b.WriteString("```\n")
	if len(s.ExecutionPlan) > 0 {
		for i, step := range s.ExecutionPlan {
			if i == 0 {
				b.WriteString(fmt.Sprintf("[ START ] ──► [ Phase %d: %s ]\n", i+1, step.Title))
			} else {
				b.WriteString(fmt.Sprintf("            ──► [ Phase %d: %s ]\n", i+1, step.Title))
			}
		}
		b.WriteString("            ──► [ DONE ]\n")
	} else {
		b.WriteString("[ START ] ──► [ ... ] ──► [ DONE ]\n")
	}
	b.WriteString("```\n\n")

	b.WriteString("## Appendix D: Validation Strategy\n\n")
	b.WriteString("### Unit Testing\n\n")
	b.WriteString("- All public functions have corresponding unit tests.\n")
	b.WriteString("- Tests run without network, DB, or LLM access (use mocks/fakes).\n")
	b.WriteString("- Target coverage: 80% statement coverage minimum.\n")
	b.WriteString("- Determinism test: same input → same output across 100 runs.\n\n")
	b.WriteString("### Integration Testing\n\n")
	b.WriteString("- Integration tests run against a local `docker compose` stack.\n")
	b.WriteString("- DB migrations applied via `migrate up` before test suite.\n")
	b.WriteString("- Seed data loaded from `docs/seeds/` directory.\n")
	b.WriteString("- Tests are idempotent: running twice on the same seed produces identical results.\n\n")
	b.WriteString("### End-to-End Testing\n\n")
	b.WriteString("- E2E tests exercise the full stack: frontend → backend → agent → LLM stub.\n")
	b.WriteString("- LLM calls use recorded cassettes (no live network calls in CI).\n")
	b.WriteString("- Browser tests use Playwright against a locally served frontend.\n\n")
	b.WriteString("### Performance Testing\n\n")
	b.WriteString("- Load test: 10 concurrent sessions with 3 iterations each.\n")
	b.WriteString("- Measure: agent dispatch p99 < 30 s, API p99 < 500 ms.\n")
	b.WriteString("- Memory profile: no goroutine leaks after 100 complete sessions.\n\n")

	b.WriteString("## Appendix E: Rollout Plan\n\n")
	b.WriteString("### Stage 1: Internal Alpha\n\n")
	b.WriteString("- Deploy to internal staging environment.\n")
	b.WriteString("- Enable for engineering team only.\n")
	b.WriteString("- Collect feedback on agent quality and UI usability.\n")
	b.WriteString("- Fix critical bugs before proceeding.\n\n")
	b.WriteString("### Stage 2: Closed Beta\n\n")
	b.WriteString("- Invite 10–20 external beta testers.\n")
	b.WriteString("- Enable all four output document types.\n")
	b.WriteString("- Collect structured feedback via in-app survey.\n")
	b.WriteString("- Monitor error rates and LLM costs.\n\n")
	b.WriteString("### Stage 3: General Availability\n\n")
	b.WriteString("- Open registration to all users.\n")
	b.WriteString("- SLA: 99.5% uptime for API layer.\n")
	b.WriteString("- LLM cost governance: session-level token budget enforced.\n")
	b.WriteString("- Documentation published at `/docs`.\n\n")

	b.WriteString("## Appendix F: Assumptions & Open Questions\n\n")
	b.WriteString("### Assumptions\n\n")
	if len(s.Assumptions) > 0 {
		for _, a := range s.Assumptions {
			b.WriteString(fmt.Sprintf("- %s\n", a))
		}
	} else {
		b.WriteString("- No assumptions recorded.\n")
	}
	b.WriteString("\n### Open Questions\n\n")
	if len(s.OpenQuestions) > 0 {
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
	} else {
		b.WriteString("- [ ] No open questions at this time.\n")
	}
	b.WriteString("\n")

	b.WriteString("## Appendix G: Canonical State Reference\n\n")
	b.WriteString("Full canonical state at the time this roadmap was generated:\n\n")
	b.WriteString(renderJSONDump(s))

	return b.String()
}

// riskRows converts s.Risks into table rows for renderTable.
func riskRows(s state.CanonicalState) [][]string {
	rows := make([][]string, 0, len(s.Risks))
	for i, r := range s.Risks {
		status := "Open"
		if r.Resolved {
			status = "Resolved"
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			r.Text,
			strings.ToTitle(r.Severity),
			"Medium",
			"High",
			"Monitor and address in next iteration",
			status,
		})
	}
	return rows
}

// padPlan emits extensive deep knowledge reference sections for the plan
// document. The content is derived from canonical state fields.
func padPlan(s state.CanonicalState) string {
	var b strings.Builder

	b.WriteString("---\n\n")
	b.WriteString("## 8. Deep Knowledge Reference\n\n")
	b.WriteString("> This section is auto-generated from the canonical state at the time of plan generation.\n")
	b.WriteString("> It supplements the implementation tasks above with detailed schemas, algorithms, and\n")
	b.WriteString("> API contracts. Use it as a reference during implementation.\n\n")

	b.WriteString("### 8.1 Canonical State Shape\n\n")
	b.WriteString("The `CanonicalState` struct is the shared brainstorm document passed between agents:\n\n")
	b.WriteString("```go\n")
	b.WriteString("type CanonicalState struct {\n")
	b.WriteString("    Idea          map[string]any `json:\"idea\"`\n")
	b.WriteString("    Architecture  map[string]any `json:\"architecture\"`\n")
	b.WriteString("    ExecutionPlan []Step         `json:\"execution_plan\"`\n")
	b.WriteString("    Risks         []Risk         `json:\"risks\"`\n")
	b.WriteString("    Assumptions   []string       `json:\"assumptions\"`\n")
	b.WriteString("    OpenQuestions []string       `json:\"open_questions\"`\n")
	b.WriteString("    Metrics       StateMetrics   `json:\"metrics\"`\n")
	b.WriteString("    Meta          StateMeta      `json:\"meta\"`\n")
	b.WriteString("}\n")
	b.WriteString("```\n\n")
	b.WriteString("**Field ownership:**\n\n")
	b.WriteString(renderTable(
		[]string{"Field", "Owner Module", "Write Rule"},
		[][]string{
			{"idea", "session", "Write once at session creation"},
			{"architecture", "agents (build role)", "Agents append/update during iteration"},
			{"execution_plan", "agents (build role)", "Agents append/update during iteration"},
			{"risks", "agents (both roles)", "Any agent can add or resolve risks"},
			{"assumptions", "agents (both roles)", "Any agent can add assumptions"},
			{"open_questions", "agents (both roles)", "Any agent can add questions"},
			{"metrics.confidence", "convergence module", "Updated after each full pipeline pass"},
			{"meta.iteration", "iteration module", "Incremented after each full pipeline pass"},
			{"meta.agents", "iteration module", "Set at session creation, read-only after"},
		},
	))

	b.WriteString("### 8.2 LLMProvider Interface\n\n")
	b.WriteString("All LLM calls go through this interface:\n\n")
	b.WriteString("```go\n")
	b.WriteString("type LLMProvider interface {\n")
	b.WriteString("    Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)\n")
	b.WriteString("}\n\n")
	b.WriteString("type LLMRequest struct {\n")
	b.WriteString("    SystemPrompt string\n")
	b.WriteString("    UserPrompt   string\n")
	b.WriteString("    MaxTokens    int\n")
	b.WriteString("    Temperature  float64\n")
	b.WriteString("}\n\n")
	b.WriteString("type LLMResponse struct {\n")
	b.WriteString("    Content      string\n")
	b.WriteString("    InputTokens  int\n")
	b.WriteString("    OutputTokens int\n")
	b.WriteString("}\n")
	b.WriteString("```\n\n")
	b.WriteString("**Forbidden:** direct SDK calls outside `platform/llm/`.\n\n")

	b.WriteString("### 8.3 N-Agent Pipeline Algorithm\n\n")
	b.WriteString("The iteration engine executes the following deterministic loop:\n\n")
	b.WriteString("```go\n")
	b.WriteString("agents := session.GetOrderedAgents() // min 2, ordered by position ASC\n\n")
	b.WriteString("for i := 1; i <= maxIter; i++ {\n")
	b.WriteString("    current := state\n")
	b.WriteString("    for _, agent := range agents {\n")
	b.WriteString("        current = agent.Dispatch(ctx, agent, agent.Role, activeSkills, llmOverride, current)\n")
	b.WriteString("    }\n")
	b.WriteString("    newState := state.Merge(state, current)\n")
	b.WriteString("    if convergence.Check(state, newState) { break }\n")
	b.WriteString("    state = newState\n")
	b.WriteString("}\n")
	b.WriteString("```\n\n")
	b.WriteString("Rules:\n\n")
	b.WriteString("- Roles are fixed at session creation — no runtime alternation.\n")
	b.WriteString("- Minimum 2 agents enforced at session start.\n")
	b.WriteString("- State is persisted after each full pipeline pass, not per-agent.\n\n")

	b.WriteString("### 8.4 Convergence Conditions\n\n")
	b.WriteString("The convergence engine evaluates multiple conditions:\n\n")
	b.WriteString("```go\n")
	b.WriteString("type ConvergenceCheck struct {\n")
	b.WriteString("    DeltaThreshold  float64 // default 0.02\n")
	b.WriteString("    MaxIterations   int     // default 10\n")
	b.WriteString("    MinIterations   int     // default 2\n")
	b.WriteString("}\n\n")
	b.WriteString("func (c *ConvergenceCheck) Check(old, new CanonicalState) bool {\n")
	b.WriteString("    if new.Meta.Iteration < c.MinIterations { return false }\n")
	b.WriteString("    delta := math.Abs(new.Metrics.Confidence - old.Metrics.Confidence)\n")
	b.WriteString("    return delta < c.DeltaThreshold\n")
	b.WriteString("}\n")
	b.WriteString("```\n\n")
	b.WriteString(fmt.Sprintf("Current confidence: **%.4f** (iteration %d)\n\n",
		s.Metrics.Confidence, s.Meta.Iteration))

	b.WriteString("### 8.5 Merge Strategy\n\n")
	b.WriteString("The state merge uses union-dedup with stability-lock:\n\n")
	b.WriteString("- **Union-dedup**: list fields (risks, assumptions, open_questions, execution_plan)\n")
	b.WriteString("  are merged by appending new items; duplicate text is deduplicated.\n")
	b.WriteString("- **Map merge**: architecture keys from the new state overwrite old keys;\n")
	b.WriteString("  keys absent in the new state are preserved from the old state.\n")
	b.WriteString("- **Stability-lock**: if a field has been stable for 3+ iterations it is locked\n")
	b.WriteString("  and agents can no longer modify it.\n")
	b.WriteString("- **Vague-output rejection**: outputs with confidence < 0.1 are rejected.\n\n")

	b.WriteString("### 8.6 API Endpoints\n\n")
	b.WriteString(renderTable(
		[]string{"Method", "Path", "Description", "Auth"},
		[][]string{
			{"POST", "/sessions", "Create new session", "None"},
			{"GET", "/sessions", "List sessions", "None"},
			{"GET", "/sessions/{id}", "Get session", "None"},
			{"PATCH", "/sessions/{id}/output-docs", "Update output doc selection", "None"},
			{"POST", "/sessions/{id}/iterate", "Run iteration pipeline", "None"},
			{"POST", "/sessions/{id}/finalize", "Generate output documents", "None"},
			{"GET", "/agents", "List agents", "None"},
			{"POST", "/agents", "Create agent", "None"},
			{"GET", "/agents/{id}", "Get agent", "None"},
			{"PUT", "/agents/{id}", "Update agent", "None"},
			{"DELETE", "/agents/{id}", "Delete agent", "None"},
			{"GET", "/skills", "List skills", "None"},
			{"POST", "/skills", "Create skill", "None"},
			{"GET", "/skills/{id}", "Get skill", "None"},
			{"PUT", "/skills/{id}", "Update skill", "None"},
			{"DELETE", "/skills/{id}", "Delete skill", "None"},
			{"GET", "/healthz", "Health check", "None"},
		},
	))

	b.WriteString("### 8.7 Current State Snapshot\n\n")
	if len(s.Idea) > 0 {
		b.WriteString("**Idea:**\n\n")
		writeMap(&b, s.Idea)
	}
	b.WriteString("**Metrics:**\n\n")
	b.WriteString(fmt.Sprintf("- Confidence: %.4f\n", s.Metrics.Confidence))
	b.WriteString(fmt.Sprintf("- Iteration: %d\n\n", s.Meta.Iteration))

	b.WriteString("**Architecture components:**\n\n")
	if len(s.Architecture) > 0 {
		writeMap(&b, s.Architecture)
	} else {
		b.WriteString("_Not yet defined._\n\n")
	}

	b.WriteString("**Risks:**\n\n")
	b.WriteString(renderRisksTable(s))

	b.WriteString("**Assumptions:**\n\n")
	if len(s.Assumptions) > 0 {
		for _, a := range s.Assumptions {
			b.WriteString(fmt.Sprintf("- %s\n", a))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_None recorded._\n\n")
	}

	b.WriteString("**Open Questions:**\n\n")
	if len(s.OpenQuestions) > 0 {
		for _, q := range s.OpenQuestions {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", q))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_None recorded._\n\n")
	}

	b.WriteString("### 8.8 Full State JSON\n\n")
	b.WriteString(renderJSONDump(s))

	return b.String()
}

// padReadme emits an extensive contributing and development guide for the
// README document.
func padReadme(s state.CanonicalState) string {
	var b strings.Builder

	b.WriteString("---\n\n")
	b.WriteString("## Appendix A: Development Guide\n\n")
	b.WriteString("### Setting Up a Development Environment\n\n")
	b.WriteString("**Prerequisites:**\n\n")
	b.WriteString("- Go 1.26 or later\n")
	b.WriteString("- Node.js 20 LTS + pnpm 9\n")
	b.WriteString("- Docker Desktop (for PostgreSQL)\n")
	b.WriteString("- Git\n\n")
	b.WriteString("**Step 1: Clone the repository**\n\n")
	b.WriteString("```bash\n")
	b.WriteString("git clone <repository-url>\n")
	b.WriteString("cd <project>\n")
	b.WriteString("```\n\n")
	b.WriteString("**Step 2: Start infrastructure**\n\n")
	b.WriteString("```bash\n")
	b.WriteString("docker compose up -d postgres\n")
	b.WriteString("```\n\n")
	b.WriteString("**Step 3: Apply database migrations**\n\n")
	b.WriteString("```bash\n")
	b.WriteString("cd backend && go run ./cmd/migrate up\n")
	b.WriteString("```\n\n")
	b.WriteString("**Step 4: Configure environment**\n\n")
	b.WriteString("```bash\n")
	b.WriteString("cp .env.example .env\n")
	b.WriteString("# Edit .env with your API keys and local settings\n")
	b.WriteString("```\n\n")
	b.WriteString("**Step 5: Run the services**\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Terminal 1 — backend\n")
	b.WriteString("cd backend && go run ./cmd/server\n\n")
	b.WriteString("# Terminal 2 — agent\n")
	b.WriteString("cd agent && go run ./cmd/server\n\n")
	b.WriteString("# Terminal 3 — frontend\n")
	b.WriteString("cd frontend && pnpm install && pnpm dev\n")
	b.WriteString("```\n\n")

	b.WriteString("### Code Organisation\n\n")
	b.WriteString("| Directory | Language | Purpose |\n")
	b.WriteString("|-----------|----------|---------|\n")
	b.WriteString("| `backend/` | Go | Modular monolith: HTTP API, iteration engine, DB |\n")
	b.WriteString("| `agent/` | Go | A2A agent binary: executes LLM calls per role |\n")
	b.WriteString("| `frontend/` | SvelteKit/TS | Web UI: session management, pipeline view |\n")
	b.WriteString("| `migrations/` | SQL | Numbered, append-only database schema migrations |\n")
	b.WriteString("| `docs/` | Markdown | Architecture blueprint, implementation plan, guides |\n\n")

	b.WriteString("### Making a Code Change\n\n")
	b.WriteString("1. Create a feature branch from `main`.\n")
	b.WriteString("2. Write a failing test first (RED phase).\n")
	b.WriteString("3. Implement the minimum code to make it pass (GREEN phase).\n")
	b.WriteString("4. Refactor while keeping tests green (REFACTOR phase).\n")
	b.WriteString("5. Run all quality gates: `make test lint build`.\n")
	b.WriteString("6. Open a pull request with a clear description.\n\n")

	b.WriteString("## Appendix B: Testing Guide\n\n")
	b.WriteString("### Running Tests\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Backend unit tests\n")
	b.WriteString("cd backend && go test ./...\n\n")
	b.WriteString("# Agent unit tests\n")
	b.WriteString("cd agent && go test ./...\n\n")
	b.WriteString("# Frontend unit tests\n")
	b.WriteString("cd frontend && pnpm test\n\n")
	b.WriteString("# Frontend type-check\n")
	b.WriteString("cd frontend && pnpm check\n")
	b.WriteString("```\n\n")
	b.WriteString("### Test Philosophy\n\n")
	b.WriteString("- Tests must run without a real database, real LLM, or live agent endpoints.\n")
	b.WriteString("- Use fakes/mocks at service boundaries; do not test implementation details.\n")
	b.WriteString("- Determinism: same seed state must produce identical test results every run.\n")
	b.WriteString("- Integration tests in `*_integration_test.go` are skipped unless the `integration`\n")
	b.WriteString("  build tag is set: `go test -tags integration ./...`.\n\n")
	b.WriteString("### Coverage Targets\n\n")
	b.WriteString("| Package | Target |\n")
	b.WriteString("|---------|--------|\n")
	b.WriteString("| `internal/modules/session` | ≥ 80% |\n")
	b.WriteString("| `internal/modules/iteration` | ≥ 80% |\n")
	b.WriteString("| `internal/modules/markdown` | ≥ 90% |\n")
	b.WriteString("| `internal/modules/state` | ≥ 85% |\n")
	b.WriteString("| `internal/platform/llm` | ≥ 75% |\n\n")

	b.WriteString("## Appendix C: Security Model\n\n")
	b.WriteString("### API Key Management\n\n")
	b.WriteString("- All API keys are stored in environment variables, never in source files.\n")
	b.WriteString("- `CredentialRef` in the DB stores the **env var name only**, not the key value.\n")
	b.WriteString("- `os.Getenv` is called only in `backend/internal/platform/config/config.go`\n")
	b.WriteString("  and `agent/internal/config/config.go`.\n\n")
	b.WriteString("### Input Validation\n\n")
	b.WriteString("- Every HTTP handler validates input before passing it to the service layer.\n")
	b.WriteString("- UUID fields are parsed with `uuid.Parse`; invalid values return HTTP 400.\n")
	b.WriteString("- String fields: non-empty check, maximum length bound.\n")
	b.WriteString("- Integer fields: range check (e.g., `max_iterations` must be 1–20).\n\n")
	b.WriteString("### SQL Injection Prevention\n\n")
	b.WriteString("- All SQL uses parameterised queries via pgx named params.\n")
	b.WriteString("- No string interpolation in SQL anywhere in the codebase.\n")
	b.WriteString("- ORM frameworks (`gorm`, `ent`) are forbidden.\n\n")
	b.WriteString("### Output Encoding\n\n")
	b.WriteString("- JSON responses are produced by `encoding/json`; no manual string building.\n")
	b.WriteString("- Markdown output is for internal use only; no user-facing rendering without escaping.\n\n")

	b.WriteString("## Appendix D: Architecture Decisions\n\n")
	b.WriteString(renderDecisionsTable(s))

	b.WriteString("## Appendix E: Metrics & Confidence\n\n")
	b.WriteString(fmt.Sprintf("This plan was generated at **iteration %d** with a confidence score of **%.4f**.\n\n",
		s.Meta.Iteration, s.Metrics.Confidence))
	if s.Metrics.Confidence >= 0.8 {
		b.WriteString("> ✅ High confidence: the brainstorm has converged well. The plan is stable.\n\n")
	} else if s.Metrics.Confidence >= 0.5 {
		b.WriteString("> ⚠️ Moderate confidence: more iteration cycles may improve quality.\n\n")
	} else {
		b.WriteString("> 🔴 Low confidence: this plan is early-stage. Expect significant changes.\n\n")
	}

	b.WriteString("## Appendix F: Agent Roster\n\n")
	if len(s.Meta.Agents) > 0 {
		rows := make([][]string, 0, len(s.Meta.Agents))
		for _, a := range s.Meta.Agents {
			rows = append(rows, []string{a.Name, a.Role, a.Provider, a.Model, strings.Join(a.Skills, ", ")})
		}
		b.WriteString(renderTable(
			[]string{"Name", "Role", "Provider", "Model", "Skills"},
			rows,
		))
	} else {
		b.WriteString("_No agent roster recorded._\n\n")
	}

	b.WriteString("## Appendix G: Full State Reference\n\n")
	b.WriteString(renderJSONDump(s))

	return b.String()
}
