**Full checklist:** `.github/skills/security-audit/SKILL.md`

1. API keys never in source/config/logs — `CredentialRef` = env var **name** only
2. `os.Getenv()` only in `backend/internal/platform/config/config.go` and `agent/internal/config/config.go`
3. `llm_config` JSONB: `{provider, model, credential_ref}` only
4. Missing credential at startup → agent unavailable (no silent fallback)
5. HTTP handlers validate input (UUID, bounds) → 400 on violation
6. Parameterized SQL only — no string interpolation
7. Never log resolved credential values
