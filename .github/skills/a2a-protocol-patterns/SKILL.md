---
name: a2a-protocol-patterns
type: skill
description: "Enforces correct usage of github.com/a2aproject/a2a-go/v2 for backend↔agent communication in the brainstorm pipeline."
---

## Purpose

This skill governs all A2A protocol interactions in `backend/internal/platform/a2a/client.go` and `agent/internal/executor/executor.go`. The project uses the `a2a-go/v2` SDK in **message-based** mode — domain context is packed as a `DataPart` inside `a2a.SendMessageRequest`, never as a custom task schema (see `docs/PLAN.md §8.3`).

## Rules

1. **Always use `BrainstormPayload` as the single wire format.** The struct defined in `backend/internal/platform/a2a/types.go` is the sole contract between backend and agent. Fields: `Role`, `SystemPrompt`, `LLMConfig`, `State`. Never add fields outside this struct.

2. **Pack payload as `DataPart` only.** Wrap `BrainstormPayload` with `a2a.NewDataPart()` before placing it in `a2a.NewMessage()`. Do not use `TextPart` or `FilePart` for structured state.

3. **Resolve `AgentCard` before every client instantiation.** In `backend/internal/platform/a2a/client.go`, call `agentcard.DefaultResolver.Resolve(ctx, agent.Endpoint)` then `a2aclient.NewFromCard(ctx, card)`. Never hardcode an A2A endpoint URL directly in `SendPayload`.

4. **Retry only on transient errors.** In `SendPayload`, retry on 5xx and timeout; fail immediately on 4xx. Reference: `docs/PLAN.md §4 (Platform: A2A Layer)` validation criteria.

5. **Extract state only from `Artifact.Parts` DataPart.** In `ExtractStateFromResult`, walk `result.Artifact.Parts`, find the `DataPart`, and unmarshal into `CanonicalState`. Do not parse raw text content.

6. **Emit the canonical 4-event sequence in `executor.go`.** Every `BrainstormExecutor.Execute` must yield: `NewSubmittedTask` → `NewStatusUpdateEvent(Working)` → `NewArtifactEvent(DataPart(updatedState))` → `NewStatusUpdateEvent(Completed)`. `Cancel` must yield `TaskStateCanceled`.

7. **Agent binary receives a fully assembled `SystemPrompt`.** The agent (`agent/internal/executor/executor.go`) must not assemble skill fragments or resolve credentials — the backend's `BuildSystemPrompt` and tiered resolver do this before dispatch. The agent is stateless with respect to skills and credential refs.

## Anti-Patterns

- **Do NOT call `client.SendMessage` with a `TextPart` payload** — structured state cannot be reliably unmarshalled from text; always use `DataPart` as specified in `docs/PLAN.md §8.3`.

- **Do NOT import `backend` module from `agent` module** — the two are separate Go modules in `go.work`. The `LLMProvider` interface is intentionally duplicated in `agent/internal/llm/` to preserve module isolation.

- **Do NOT skip the `AgentCard` resolution step** — constructing `a2aclient` directly from a URL bypasses capability negotiation and breaks the A2A discovery contract.

- **Do NOT add business logic in `executor.go`** — the executor's only job is to extract `BrainstormPayload`, call `LLMProvider.Generate`, and emit events. Role logic, merge logic, and convergence checks live in the backend modules.

## Checklist

```
[ ] BrainstormPayload is the only wire struct; no fields added outside types.go
[ ] Payload wrapped with a2a.NewDataPart() before SendMessageRequest
[ ] AgentCard resolved via agentcard.DefaultResolver before client creation
[ ] Retry logic applied on 5xx/timeout only; 4xx causes immediate failure
[ ] State extracted from Artifact.Parts DataPart (not text content)
[ ] executor.go emits all 4 events in order: Submitted → Working → Artifact → Completed
[ ] Cancel emits TaskStateCanceled
[ ] agent/internal/llm/ defines its own LLMProvider copy (no cross-module import)
[ ] Agent binary receives assembled SystemPrompt; no skill or credential resolution inside executor
```
