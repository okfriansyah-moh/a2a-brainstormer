---
name: llm-provider-abstraction
type: skill
description: "Enforces the LLMProvider interface, tiered config resolver, and credential-via-env-ref rules across backend/internal/platform/llm/ and agent/internal/llm/."
---

## Purpose

This skill governs `backend/internal/platform/llm/provider.go`, `resolver.go`, `copilot.go`, and their mirror in `agent/internal/llm/`. The project requires that all LLM calls go through the `LLMProvider` interface — never directly to a provider SDK — and that API credentials are resolved at call time from environment variable references, never stored as raw values. Rules come from `docs/PLAN.md §8.2` and `§3 (Task 3)`.

## Rules

1. **All LLM calls must go through `LLMProvider.Generate`.** No module (agent executor, iteration engine, or any handler) may import or call a Copilot/Claude SDK directly. Only `CopilotProvider` and future `ClaudeProvider` implementations in `platform/llm/` and `agent/internal/llm/` may do so.

2. **`LLMProvider` interface is defined in `platform/llm/provider.go` with exactly these types:**
   ```go
   type LLMProvider interface {
       Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
   }
   type LLMRequest struct { SystemPrompt string; UserMessage string; Temperature float64 }
   type LLMResponse struct { Content string; FinishReason string; TokensUsed int }
   ```
   Do not add methods to the interface; keep it minimal.

3. **`CredentialRef` is an env var name, never a raw API key.** `LLMConfig.CredentialRef` stores a string like `"CLAUDE_API_KEY"`. The actual key is resolved at call time via `ResolveKey(credentialRef string)` which calls `os.Getenv(credentialRef)`. Never store a literal API key in `LLMConfig`, DB, or config files.

4. **`ResolveKey` returns an error if the env var is absent.** No silent fallback. If `os.Getenv(credentialRef) == ""`, return a descriptive error. The calling agent service marks the agent as `unavailable` (see `agent/service.go:CheckAvailability`).

5. **Tiered resolver priority: session override → agent-level → global default.** Implement `Resolve(global, agentLevel, sessionOverride *LLMConfig) LLMConfig` in `resolver.go`. Apply field-by-field override: a non-empty field in a higher-priority config wins. A nil pointer means "not set at this tier".

6. **`LLMProvider` interface is duplicated in `agent/internal/llm/`.** The `agent` module is a separate Go module and must not import from `backend`. Copy the interface definition rather than creating a shared dependency. Both copies must remain structurally identical.

7. **`CopilotProvider` uses low temperature for determinism.** Set `Temperature` to a low fixed value (≤ 0.2) when no caller-provided value is set. Use structured JSON schema prompts. This is required to reduce Copilot inconsistency (see `docs/A2A-agent-Brainstorm.md §17.4`).

8. **`GetGlobalLLMProvider`, `GetGlobalLLMModel`, `GetGlobalLLMCredentialRef` are the only env var accessors for LLM config.** All callers must use these from `backend/internal/platform/config/config.go`. Never call `os.Getenv` for LLM config outside `config.go` and `resolver.go:ResolveKey`.

## Anti-Patterns

- **Do NOT store a raw API key anywhere except a live environment variable.** No `config.yaml`, no `.env` committed to source, no plaintext in the DB `llm_config` JSONB column. The column stores only `{provider, model, credential_ref}`.

- **Do NOT add a `ClaudeProvider` stub that returns empty responses.** Future providers must either be fully implemented or not compiled into the binary. An empty `ClaudeProvider` silently produces zero-quality output without error.

- **Do NOT let `LLMProvider` interface grow beyond `Generate`.** Adding `Stream`, `Embed`, or other methods breaks the minimal contract and forces all implementations to change. Keep the interface at one method.

- **Do NOT call `os.Getenv` for credentials inside `executor.go` or any module outside `config.go`/`resolver.go`.** Centralizing env var access in these two files is the project's primary mechanism for auditing credential exposure.

## Checklist

```
[ ] LLMProvider interface has exactly one method: Generate(ctx, LLMRequest) (LLMResponse, error)
[ ] No Copilot/Claude SDK import outside platform/llm/ and agent/internal/llm/
[ ] LLMConfig.CredentialRef stores env var name only (e.g. "CLAUDE_API_KEY")
[ ] ResolveKey returns error on empty env var; no silent fallback
[ ] Resolve() applies field-by-field tiered priority: session > agent > global
[ ] agent/internal/llm/ has its own LLMProvider copy; no cross-module import from backend
[ ] CopilotProvider uses Temperature ≤ 0.2 as default for determinism
[ ] All LLM env var access goes through platform/config/config.go getters
[ ] llm_config JSONB column contains only {provider, model, credential_ref}
```
