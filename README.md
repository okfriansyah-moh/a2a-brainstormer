# a2a-brainstorm

Deterministic multi-agent brainstorming platform that turns an initial idea into two engineering artifacts:

- `architecture.md`
- `roadmap.md`

The system uses an ordered N-agent pipeline (minimum 2 agents), A2A message-based communication, and convergence rules to iteratively refine output quality.

## What This Project Is

- A structured design workspace, not a chat app
- A Go modular monolith backend with vertical-slice modules
- Multiple Go agent services connected through `github.com/a2aproject/a2a-go/v2`
- A SvelteKit frontend for session orchestration and side-by-side agent outputs
- PostgreSQL-backed canonical state, agent registry, and iteration history

## Core Objective

Input a product idea, run deterministic multi-agent iterations, detect convergence, and emit final design artifacts.

Pipeline intent:

`idea -> ordered agent passes -> merge -> convergence check -> markdown artifacts`

## High-Level Architecture

```text
Frontend (SvelteKit)
        |
        v
Backend (Go 1.26 modular monolith)
        |
        v
A2A agents (Go services, role-based dispatch)
        |
        v
LLM providers (Copilot-first, Claude-ready)

PostgreSQL stores canonical state and iteration records
Markdown module generates architecture.md and roadmap.md
```

## Technical Stack

- Backend: Go 1.26
- A2A SDK: `github.com/a2aproject/a2a-go/v2`
- Frontend: SvelteKit + TypeScript + TailwindCSS
- Database: PostgreSQL 16 (`pgx/v5`, `sqlc`)
- Deployment: Docker + docker-compose

## Architectural Rules (Summary)

- Modular monolith with strict module boundaries
- Vertical slice per module (`handler.go`, `service.go`, `repository.go`, `model.go`)
- No cross-module internal imports
- No direct LLM SDK calls outside provider boundary (`LLMProvider` interface)
- Credential refs store env var names only (never raw keys)
- Deterministic and idempotent iteration behavior is mandatory

## Planned Backend Modules

- `session`: session lifecycle and setup
- `iteration`: orchestration loop and dispatch sequence
- `agent`: registry, role config, and A2A dispatch integration
- `state`: canonical state model, validation, merge
- `convergence`: stop-condition engine
- `markdown`: final artifact generation

## Current Status

- Blueprint complete: `docs/A2A-agent-Brainstorm.md`
- Implementation plan complete: `docs/PLAN.md`
- Governance in place: `AGENTS.md` and `.github/copilot-instructions.md`
- Implementation tasks (Task 1-15) are defined and ready to execute

## Repository Docs

- Architecture blueprint: `docs/A2A-agent-Brainstorm.md`
- Execution plan: `docs/PLAN.md`
- Agent and skill governance: `AGENTS.md`
- Copilot execution rules: `.github/copilot-instructions.md`

## Next Step

Start implementation from Task 1 in `docs/PLAN.md` and follow task ordering through Task 15 with validation gates at each phase.
