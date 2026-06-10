**Full table:** `.github/skills/code-quality/SKILL.md` · `.github/skills/security-audit/SKILL.md`| Category | Forbidden |
| --- | --- |
| Architecture | Microservices between modules, inter-module RPC, shared mutable globals |
| Database | ORM (`gorm`, `ent`), SQL concat, driver imports in `internal/modules/` |
| LLM | Direct SDK calls in `internal/modules/` or `agent/internal/executor/` |
| Config | Hardcoded keys/ports/models; `os.Getenv` outside config files |
| State | Non-deterministic IDs from timestamps; per-agent mutable globals |
| Naming | Task-code filenames (`phase4.go`), single-letter files |
| Go format | Stray `package` line before doc comment on line 1 — check every new `.go` file |
| UI (v1.1+) | Hardcoded hex colors; new `/agents`/`/skills` pages; deprecated `AgentPanel`/`StateView`/`Timeline`; WebSocket for SSE; preview UI outside `PipelineStage` |