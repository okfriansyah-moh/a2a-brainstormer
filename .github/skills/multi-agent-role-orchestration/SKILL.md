---
name: multi-agent-role-orchestration
type: skill
description: "Enforces ordered N-agent pipeline dispatch, role assignment rules, and sequential state threading through the iteration engine."
---

## Purpose

This skill governs `backend/modules/iteration/engine.go`, `backend/modules/agent/role.go`, and `backend/modules/agent/client.go`. The project uses a fixed-order, sequential pipeline — roles are assigned at session creation, not at runtime. The original two-agent alternation rule (`i%2 parity`) was superseded by the dynamic N-agent model described in `docs/PLAN.md §8.4`.

## Rules

1. **Roles are fixed at session creation.** `backend/modules/session/service.go:CreateSession` assigns roles from `req.RoleOverrides` (if provided) or `agent.DefaultRoles(len(agentIDs))`. Roles must not change between iterations. See `docs/PLAN.md §8.4`: "Roles are fixed at session creation — no runtime alternation."

2. **Each agent receives the previous agent's output, not the original state.** In `iteration/engine.go`, the inner loop passes `current` (updated by the preceding agent) to each subsequent agent. The base `state` is only used for `convergence.Check` comparison after a full pipeline pass.

3. **Ordered pipeline by `session_agents.position ASC`.** `GetOrderedAgents(sessionID)` returns agents sorted by `position`. The iteration engine must iterate this slice in order; do not sort by agent ID or name.

4. **State is persisted once per full pipeline pass, not per agent.** Call `persistState` after all agents in a pass have run, not inside the agent dispatch loop. Reference: `docs/PLAN.md §8.4`.

5. **`agent.DefaultRoles(n)` governs role distribution.** For new sessions without `role_overrides`, use `modules/agent/role.go:DefaultRoles(agentCount)`. Do not hardcode `RoleBuilder` / `RoleReviewer` assignments outside this function.

6. **`ValidRole(r Role)` must be called before dispatch.** In `agent/service.go:RegisterAgent` and session creation, validate every role against the allowlist: `RoleBuilder`, `RoleReviewer`, `RoleRefiner`, `RoleDevilsAdvocate` (see `docs/PLAN.md §8.13`).

7. **`meta.agents` in `CanonicalState` reflects the session's agent list.** After each iteration, `state.Meta.Agents` must list all session agents with their assigned roles and active skill names. Fixed keys `agentA`/`agentB` are forbidden — use the dynamic slice.

## Anti-Patterns

- **Do NOT use `i%2` parity to assign roles at runtime.** The original blueprint draft described alternation; the final spec (`docs/PLAN.md §8.4`) explicitly prohibits this: "Roles are fixed at session creation — no runtime alternation."

- **Do NOT pass the original `state` to agent N>1 in a pipeline pass.** Each agent must receive the output of the agent before it, creating a cumulative refinement chain.

- **Do NOT allow fewer than 2 agents per session.** `session/service.go:CreateSession` must reject `CreateSessionRequest` with `len(agent_ids) < 2` with a `400` error. This is enforced in both backend validation and the frontend `AgentSelector.svelte`.

- **Do NOT persist state inside the agent dispatch loop.** Partial-pass state must not be written to the DB; only a complete pipeline pass result is persisted.

## Checklist

```
[ ] Roles assigned in CreateSession (RoleOverrides or DefaultRoles), never inside engine loop
[ ] GetOrderedAgents returns agents sorted by position ASC
[ ] Inner loop passes `current` (prev agent output) to each next agent
[ ] State.Merge called once per pass on accumulated `current`; result persisted once
[ ] convergence.Check compares base `state` to `newState` (full-pass result)
[ ] meta.agents populated with dynamic agent list; no agentA/agentB keys
[ ] ValidRole called on every role value before dispatch
[ ] CreateSession rejects requests with fewer than 2 agent IDs
```
