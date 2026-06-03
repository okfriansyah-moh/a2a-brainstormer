# AI hub — provider-agnostic config (approved)

**Date:** 2026-06-03  
**Status:** Implemented  
**Decision:** Option C — neutral `ai/` hub with `.github/` symlinks + composed IDE adapters

## Problem

IDE-specific trees (`.cursor/`, `.github/`, `.claude/`) duplicate agents, skills, and global rules. Edits drift; teams using multiple assistants maintain several copies.

This is **orthogonal** to runtime `LLMProvider` in Go (Copilot / Claude / OpenCode). Here we only standardize **repository AI configuration**.

## Solution

```
ai/                          ← canonical (edit here)
├── instructions.md          ← global invariants
├── agents/*.md
├── skills/*/SKILL.md
└── prompts/*.md

.github/                     ← Copilot-native symlinks
├── copilot-instructions.md → ../ai/instructions.md
├── agents/                 → ../ai/agents/
├── skills/                 → ../ai/skills/
└── prompts/                → ../ai/prompts/

.cursor/                     ← generated (gitignored)
├── rules/project-invariants.mdc
├── agents/
└── skills/

CLAUDE.md, .claude/skills/  ← generated (gitignored)
```

## Portable frontmatter

**Agents** (`ai/agents/`): `name`, `description`, optional `readonly`, `skills[]`, `subagents[]`

**Skills** (`ai/skills/`): `name`, `description`, optional `invocation: manual|auto`

**Prompts** (`ai/prompts/`): same as skills; always `invocation: manual`

`scripts/ai/compose.py` maps to Cursor (`disable-model-invocation`, `model`, `readonly`, `is_background`) and copies skills for Claude Code.

## Operations

| Command | Purpose |
| ------- | ------- |
| `./scripts/ai/link-github.sh` | Create / refresh `.github/` symlinks |
| `python3 scripts/ai/compose.py` | Generate `.cursor/`, `CLAUDE.md`, `.claude/skills/` |
| `./scripts/ai/setup.sh` | Both of the above |

Run after clone and after any `ai/` edit (if using Cursor or Claude Code).

## Constraints

- Symlinks require Unix-style checkout (`git clone` on Windows: enable Developer Mode or `core.symlinks=true`).
- Copilot filename `copilot-instructions.md` is retained as a symlink target name only — content is vendor-neutral in `ai/instructions.md`.
- Backend `SKILL_BUNDLE_PATHS` defaults use `ai/skills/...` paths.

## References

- [AGENTS.md](../../AGENTS.md) — registry and governance
- [ai/README.md](../../ai/README.md) — hub overview
