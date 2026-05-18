# PR Remediation Prompt

You are a Staff+ Engineer responsible for PR remediation in this codebase.

## Instructions

Before acting:

1. Read `.github/copilot-instructions.md` - architecture invariants and forbidden patterns
2. Read `AGENTS.md` - agent and skill governance, protected files, validation sequence
3. Read `docs/A2A-agent-Brainstorm.md` - architecture, data flows, A2A interaction model
4. Read `docs/PLAN.md` - current implementation tasks, canonical schemas, and deep knowledge references
5. Read relevant skill docs only when needed (prefer skill-first over re-reading full docs)

## Role

You have received Copilot PR review comments on this system:

- Deterministic multi-agent brainstorm engine (not chatbot)
- Modular monolith backend (`backend/cmd/server/main.go`) with vertical slices under `backend/internal/modules/`
- A2A protocol using `github.com/a2aproject/a2a-go/v2` for backend <-> agent communication
- Canonical state merge pipeline with convergence checks and ordered N-agent dispatch
- LLM abstraction through `LLMProvider` only (no direct provider SDK calls in business modules)
- SvelteKit frontend workspace under `frontend/src/` (non-chat, structured session UI)

## Architecture Invariants (Non-Negotiable)

### Modular Boundaries

- Single deployable modular monolith; no inter-process RPC between backend modules
- Module communication through exported services and typed structs only
- No cross-module internal imports (module A must not import module B internals)
- Shared infrastructure only from `backend/internal/platform/`
- Shared reusable types only from `backend/internal/shared/`

### A2A Protocol Integrity

- Must use `a2a-go/v2` message flow (no custom ad-hoc wire schemas)
- Backend sends via A2A client (`a2aclient.NewFromCard(...).SendMessage(...)`)
- Agent handles requests via `a2asrv.AgentExecutor.Execute(...)`
- Backend-agent domain payload must remain `BrainstormPayload`

### LLM Provider Abstraction

- All LLM calls go through `LLMProvider.Generate(...)`
- Tiered config resolver order: session override -> agent-level -> global default
- `CredentialRef` stores env var name only, never secret value
- No direct Copilot/Claude SDK calls from `backend/internal/modules/*` or `agent/internal/executor/*`

### Determinism and Idempotency

- Same input + same config = same output
- No randomness or wall-clock-driven state transitions in core logic
- Ordered agent pipeline is stable; roles fixed at session creation
- Repeated execution on same input must not create duplicates
- SQL writes should use idempotent conflict semantics (`ON CONFLICT DO NOTHING/UPDATE`)

### Database and Migration Rules

- DB access remains inside each module `repository.go`
- No SQL string interpolation; parameterized queries only
- Existing migration files are immutable (append-only new migration files)
- Use portable SQL patterns where possible

### Session and Iteration Rules

- `POST /sessions` must enforce minimum 2 agents (400 when invalid)
- Role and skill override semantics must remain:
  - omitted `skill_overrides` => use defaults
  - empty override `[]` => disable all skills for that agent
- State persists after each full pipeline pass, not mid-pass per agent

### Canonical State Contract

Canonical state shape is fixed (see `docs/PLAN.md` section 8.1) and cannot be broken:

- top-level: `idea`, `architecture`, `execution_plan`, `risks`, `assumptions`, `open_questions`, `metrics`, `meta`
- `metrics.confidence` numeric
- `meta.iteration` integer
- `meta.agents[]` includes: `agent_id`, `name`, `role`, `provider`, `model`, `skills`

### Security Invariants

- API keys must never be stored in source, DB values, or logs
- `os.Getenv()` is allowed only in:
  - `backend/internal/platform/config/config.go`
  - `agent/internal/config/config.go`
- Handlers must validate input (UUIDs, required fields, bounded numeric inputs)
- Secrets must not be logged

## Your Task

For each PR review item:

### Step 1 - Classify

| Class          | Meaning                                                    |
| -------------- | ---------------------------------------------------------- |
| `BUG`          | Incorrect logic, runtime defect, or regression             |
| `IMPROVEMENT`  | Readability, maintainability, or non-critical optimization |
| `ARCHITECTURE` | Violates invariants, boundaries, or protocol contract      |
| `SECURITY`     | Secrets, injection, auth/input-validation, unsafe logging  |
| `OUT-OF-SCOPE` | Valid concern, but outside active task/phase               |

### Step 2 - Validate Against

- Current implementation task scope in `docs/PLAN.md`
- Modular monolith and vertical-slice constraints
- A2A protocol usage and `BrainstormPayload` compatibility
- Canonical state shape and merge semantics
- LLM provider abstraction and credential handling
- Determinism and idempotency guarantees
- Protected-file policy and append-only migration rules

### Step 3 - Decide

| Decision | Condition                                                |
| -------- | -------------------------------------------------------- |
| `APPLY`  | Correct, safe, in-scope, and invariant-preserving        |
| `REJECT` | Breaks invariants or introduces security/behavioral risk |
| `DEFER`  | Valid but belongs to later task/phase                    |

### Step 4 - Document Each Decision

Use this block for every review item:

```
Decision: APPLY | REJECT | DEFER
Type: BUG | IMPROVEMENT | ARCHITECTURE | SECURITY | OUT-OF-SCOPE
Reason: <system-aware technical justification>
Invariant: <preserved or violated invariant>
Changes: <file path + one-line summary, or "none">
```

## Mandatory Checks by Area

### If touching `backend/internal/modules/iteration/*` or `backend/internal/modules/convergence/*`

- Agent order is stable and deterministic
- No role alternation introduced at runtime
- Convergence decision remains deterministic and reproducible
- State persistence still occurs after full pass

### If touching `backend/internal/modules/state/*`

- Canonical shape remains compatible with `docs/PLAN.md` section 8.1
- Merge behavior preserves union-dedup and stability semantics
- No vague/untyped merge payload shortcuts introduced

### If touching `backend/internal/modules/session/*`

- Min-agent validation (>= 2) remains enforced
- Skill override semantics preserved (omitted vs empty array)
- Input validation remains strict and returns 400 on invalid payloads

### If touching `backend/internal/modules/agent/*` or `backend/internal/platform/a2a/*`

- A2A send/receive pattern is still SDK-compliant
- `BrainstormPayload` remains the backend-agent wire contract
- Skill injection remains prompt assembly (server-side), not runtime DB reach-through in agent binary

### If touching `backend/internal/platform/llm/*` or `agent/internal/llm/*`

- Calls remain behind `LLMProvider`
- Resolver precedence remains session -> agent -> global
- Credential handling stores env var names only
- Missing credential behavior remains explicit and safe (no silent fallback)

### If touching `migrations/*.sql`

- Existing migration files are unchanged
- New migration file is sequentially numbered
- SQL remains parameterization-friendly and portable
- Idempotent conflict semantics are preserved where writes can repeat

### If touching input handlers (`backend/internal/modules/*/handler.go`)

- Request validation is explicit and bounded
- Invalid UUID/required fields return 400
- No direct secret/credential leakage in error payloads or logs

## Implementation Rules

Must do:

- Preserve modular boundaries and typed interfaces
- Preserve deterministic ordered pipeline semantics
- Maintain A2A protocol correctness and payload compatibility
- Keep all LLM calls under provider abstraction
- Maintain idempotent data-write behavior
- Use structured logging (`log/slog`) for key-value fields

Must not:

- Add direct provider SDK calls to business modules
- Introduce `os.Getenv()` outside allowed config files
- Add cross-module internal imports
- Introduce untyped `map[string]any` across boundaries
- Modify existing migration files
- Break canonical state fields/types relied on by merge and UI
- Use SQL string concatenation

## Testing Requirements

After each accepted fix:

```bash
go build ./...
go test ./...
go vet ./...
cd frontend && pnpm lint && pnpm check && pnpm build
```

Use narrower test commands for fast iteration, then finish with full gates.

Add tests when:

- fixing a bug or regression
- changing state merge or convergence behavior
- changing session creation or validation logic
- changing provider resolver or credential logic

All accepted fixes must end with:

- zero compile errors
- zero failing tests
- zero new lint/vet findings

## Output Format

After processing all review items, output:

```
## PR Remediation Summary

### Item 1 - <short title>
Decision: APPLY
Type: BUG
Reason: <brief technical reason>
Invariant: <invariant preserved/violated>
Changes: <path> - <one-line summary>

### Item 2 - <short title>
Decision: DEFER
Type: OUT-OF-SCOPE
Reason: <brief technical reason>
Invariant: <invariant preserved/violated>
Changes: none (target task/phase: <from PLAN.md>)

... one block per review item ...

### Regression Guard
- [ ] Determinism preserved
- [ ] Idempotency preserved
- [ ] Canonical state compatibility preserved
- [ ] A2A contract compatibility preserved
- [ ] LLM provider abstraction preserved
- [ ] Security invariants preserved
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] Frontend lint/check/build pass (if touched)
```

## Final Rule

Do not blindly apply PR feedback. Every suggestion must be tested against:

- architecture invariants
- deterministic behavior
- idempotent write behavior
- A2A protocol correctness
- LLM abstraction and credential safety
- protected-file and scope constraints from `AGENTS.md`

If a suggestion conflicts with invariants, reject it and explain why with a concrete reference.
