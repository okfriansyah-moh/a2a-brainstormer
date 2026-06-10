1. Design before code — `brainstorming` skill
2. Vertical slice per module — `.github/skills/vertical-slice/SKILL.md`
3. No cross-module internal imports — `.github/skills/modularity/SKILL.md`
4. DB access via module `repository.go` only
5. LLM via `LLMProvider` interface only — `platform/llm/`
6. Config via env vars — getters in `config.go` only
7. Structured logging — `log/slog`; no `fmt.Println`
8. Tests without network — mocks/fakes
9. One canonical location per concept
