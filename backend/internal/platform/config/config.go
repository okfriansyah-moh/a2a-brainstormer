// Package config centralises ALL os.Getenv calls for the backend binary.
// No other file in the backend may call os.Getenv directly.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ── Database ─────────────────────────────────────────────────────────────────

// GetDatabaseURL returns the PostgreSQL connection string from DATABASE_URL.
// Returns an error if the variable is absent or empty — never silently falls
// back to a default connection string.
func GetDatabaseURL() (string, error) {
	v := os.Getenv("DATABASE_URL")
	if v == "" {
		return "", errors.New("DATABASE_URL environment variable is required but not set")
	}
	return v, nil
}

// ── Iteration engine ─────────────────────────────────────────────────────────

// GetMaxIterations returns the maximum number of pipeline passes before the
// engine force-stops. Defaults to 10 when MAX_ITERATIONS is unset.
func GetMaxIterations() int {
	return envInt("MAX_ITERATIONS", 10)
}

// GetConvergenceThreshold returns the minimum confidence delta below which
// the engine considers the pipeline converged. Defaults to 0.02.
func GetConvergenceThreshold() float64 {
	return envFloat("CONVERGENCE_THRESHOLD", 0.02)
}

// GetMinConfidenceFloor returns the minimum confidence score that must be
// reached before the engine is allowed to converge. The threshold is applied
// to CanonicalState.Metrics.Confidence (aggregate pipeline confidence), not an
// individual agent score. Defaults to 0.90 and is clamped to [0.0, 1.0].
// Set MIN_CONFIDENCE_FLOOR to override.
func GetMinConfidenceFloor() float64 {
	v := envFloat("MIN_CONFIDENCE_FLOOR", 0.90)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ── Global LLM defaults ───────────────────────────────────────────────────────

// GetGlobalLLMProvider returns the default LLM provider name.
// Allowed values: "copilot" | "claude" | "opencode" | "deepseek".
// Defaults to "deepseek".
func GetGlobalLLMProvider() string {
	return envString("GLOBAL_LLM_PROVIDER", "deepseek")
}

// GetGlobalLLMModel returns the default LLM model name. Defaults to "deepseek-v4-flash".
func GetGlobalLLMModel() string {
	return envString("GLOBAL_LLM_MODEL", "deepseek-v4-flash")
}

// GetGlobalLLMCredentialRef returns the env var NAME that holds the global LLM
// API key. This is a reference (env var name), never the raw key value.
// Defaults to "DEEPSEEK_API_KEY".
func GetGlobalLLMCredentialRef() string {
	return envString("GLOBAL_LLM_CREDENTIAL_REF", "DEEPSEEK_API_KEY")
}

// SetGlobalLLMConfig writes the global LLM defaults back to the runtime env.
// Empty values fall back to the same DeepSeek defaults used at startup.
// CredentialRef is derived from the provider — never accepted from API callers.
func SetGlobalLLMConfig(provider, model string) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "deepseek"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "deepseek-v4-flash"
	}
	credentialRef := DefaultGlobalCredentialRef(provider)

	os.Setenv("GLOBAL_LLM_PROVIDER", provider)
	os.Setenv("GLOBAL_LLM_MODEL", model)
	os.Setenv("GLOBAL_LLM_CREDENTIAL_REF", credentialRef)
}

// DefaultGlobalCredentialRef returns the canonical env var name for a provider's
// API key. Used when persisting global LLM settings from the settings UI.
func DefaultGlobalCredentialRef(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "copilot":
		return "COPILOT_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "claude":
		return "ANTHROPIC_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	case "opencode":
		return GetOpenCodeServerPasswordRef()
	default:
		return GetGlobalLLMCredentialRef()
	}
}

// GetLLMAPIKey resolves a CredentialRef to its actual key value at runtime.
// Returns an error if the referenced env var is absent or empty — no silent
// fallback to another provider. This is the ONLY place a resolved key may be
// read; the key itself must never be logged or stored.
func GetLLMAPIKey(credentialRef string) (string, error) {
	if credentialRef == "" {
		return "", fmt.Errorf("credentialRef is empty: cannot resolve LLM API key")
	}
	key := os.Getenv(credentialRef)
	if key == "" {
		return "", fmt.Errorf("LLM credential env var %q is not set: agent unavailable", credentialRef)
	}
	return key, nil
}

// ── Agent registry ────────────────────────────────────────────────────────────

// GetAgentEndpoints returns the comma-separated list of agent base URLs used
// in local development (e.g. "http://localhost:9090"). Returns an empty slice
// when AGENT_ENDPOINTS is unset.
func GetAgentEndpoints() []string {
	raw := os.Getenv("AGENT_ENDPOINTS")
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ── HTTP server ───────────────────────────────────────────────────────────────

// GetBackendPort returns the TCP port the HTTP server listens on. Defaults to 8080.
func GetBackendPort() string {
	return envString("BACKEND_PORT", "8080")
}

// GetOutputDir returns the filesystem directory where finalized session
// artifacts (architecture.md, roadmap.md) are written. Defaults to "output".
func GetOutputDir() string {
	return envString("OUTPUT_DIR", "output")
}

// GetIterationTimeout returns the maximum duration allowed for a single full
// iteration pipeline run (all agents × all passes). The timer is independent
// of the HTTP request lifetime so that a client disconnect cannot abort an
// in-flight LLM pipeline. Defaults to 30 minutes.
// Set ITERATION_TIMEOUT_SECONDS to override.
func GetIterationTimeout() time.Duration {
	return time.Duration(envInt("ITERATION_TIMEOUT_SECONDS", 1800)) * time.Second
}

// GetAgentCallTimeout returns the maximum wall-clock time for a single A2A agent
// call (streaming or blocking). Defaults to 30 minutes.
// Set AGENT_CALL_TIMEOUT_SECONDS to override.
func GetAgentCallTimeout() time.Duration {
	return time.Duration(envInt("AGENT_CALL_TIMEOUT_SECONDS", 1800)) * time.Second
}

// GetAgentStreamIdleTimeout returns how long a streaming agent call may go
// without receiving tokens before the backend aborts it. Defaults to 10 minutes.
// Set AGENT_STREAM_IDLE_TIMEOUT_SECONDS to override.
func GetAgentStreamIdleTimeout() time.Duration {
	return time.Duration(envInt("AGENT_STREAM_IDLE_TIMEOUT_SECONDS", 600)) * time.Second
}

// GetAgentFirstTokenTimeout returns how long the backend waits for the first
// streamed token from an agent before aborting the call. Defaults to 3 minutes.
// Set AGENT_FIRST_TOKEN_TIMEOUT_SECONDS to override.
func GetAgentFirstTokenTimeout() time.Duration {
	return time.Duration(envInt("AGENT_FIRST_TOKEN_TIMEOUT_SECONDS", 180)) * time.Second
}

// GetFinalizeTimeout returns the maximum duration allowed for one finalize
// document-generation call (deterministic scaffold + optional AI enhance pass).
// This timeout is applied per-request, independently of the server WriteTimeout.
//
// Section-sequential docs (architecture, plan) issue many LLM calls (one per
// section plus rubric repairs and coherence). The previous 10-minute default
// routinely expired before the last sections, marking the whole doc ai_fallback.
// Defaults to 60 minutes. Set FINALIZE_TIMEOUT_SECONDS to override.
func GetFinalizeTimeout() time.Duration {
	return time.Duration(envInt("FINALIZE_TIMEOUT_SECONDS", 3600)) * time.Second
}

// GetFinalizeMode returns the document-generation strategy for session
// finalize. Valid values: "deterministic", "hybrid", "ai". Defaults to
// "hybrid". Unknown values fall back to "deterministic" downstream.
// Set FINALIZE_MODE to override.
func GetFinalizeMode() string {
	return envString("FINALIZE_MODE", "hybrid")
}

// GetSkillBundlePaths returns skill file paths loaded into the AI doc-generator
// system prompt. Empty by default — set SKILL_BUNDLE_PATHS to opt in.
func GetSkillBundlePaths() []string {
	raw := envString("SKILL_BUNDLE_PATHS", "")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// defaultSkillBundlePaths is empty by default so AI document generation stays
// anchored to the user's product canonical state. The previous defaults loaded
// this repository's engineering skills (modularity, vertical-slice, etc.) and
// caused plan/architecture docs to describe the A2A Brainstorm tool instead of
// the user's idea. Set SKILL_BUNDLE_PATHS to opt in (comma-separated).
var defaultSkillBundlePaths []string

// GetDeepSeekBaseURL returns the DeepSeek API base URL.
// Reads DEEPSEEK_BASE_URL; defaults to "https://api.deepseek.com".
func GetDeepSeekBaseURL() string {
	return envString("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
}

// GetOpenAIBaseURL returns the OpenAI API base URL.
// Reads OPENAI_BASE_URL; defaults to "https://api.openai.com/v1".
// Override for Azure OpenAI or proxy deployments.
func GetOpenAIBaseURL() string {
	return envString("OPENAI_BASE_URL", "https://api.openai.com/v1")
}

// GetAnthropicBaseURL returns the Anthropic API base URL.
// Reads ANTHROPIC_BASE_URL; defaults to "https://api.anthropic.com".
func GetAnthropicBaseURL() string {
	return envString("ANTHROPIC_BASE_URL", "https://api.anthropic.com")
}

// ── OpenCode (used when GLOBAL_LLM_PROVIDER=opencode) ────────────────────────

// GetGlobalOpenCodeBaseURL returns the HTTP base URL of the OpenCode server
// that the backend will use for AI document generation.
// Defaults to "http://opencode:4096" (Docker service name). Override with
// GLOBAL_OPENCODE_BASE_URL (e.g. "http://localhost:4096" for local dev without Docker).
func GetGlobalOpenCodeBaseURL() string {
	return envString("GLOBAL_OPENCODE_BASE_URL", "http://opencode:4096")
}

// GetGlobalOpenCodeModel returns the providerID/modelID string for the OpenCode
// server request. Format: "<providerID>/<modelID>", e.g.
// "github-copilot/claude-sonnet-4.6". Defaults to that value.
// Set GLOBAL_OPENCODE_MODEL to override.
func GetGlobalOpenCodeModel() string {
	return envString("GLOBAL_OPENCODE_MODEL", "github-copilot/claude-sonnet-4.6")
}

// GetOpenCodeServerUsernameRef returns the env var NAME that holds the
// Basic-Auth username for the OpenCode server. Defaults to
// "OPENCODE_SERVER_USERNAME".
func GetOpenCodeServerUsernameRef() string {
	return envString("OPENCODE_USERNAME_REF", "OPENCODE_SERVER_USERNAME")
}

// GetOpenCodeServerPasswordRef returns the env var NAME that holds the
// Basic-Auth password for the OpenCode server. Defaults to
// "OPENCODE_SERVER_PASSWORD".
func GetOpenCodeServerPasswordRef() string {
	return envString("OPENCODE_PASSWORD_REF", "OPENCODE_SERVER_PASSWORD")
}

// GetAIDocMaxRepairs returns the maximum number of rubric-driven repair
// attempts the AI doc generator may issue per document. Clamped to [0, 5].
// Defaults to 3 — long-form (≥1000 line) documents typically require at
// least one expansion pass after the initial draft. Set AIGEN_MAX_REPAIRS to
// override.
func GetAIDocMaxRepairs() int {
	v := envInt("AIGEN_MAX_REPAIRS", 3)
	if v < 0 {
		return 0
	}
	if v > 5 {
		return 5
	}
	return v
}

// GetAIDocTemperature returns the LLM temperature used for AI document
// rewriting. Clamped to [0.0, 1.0]. Defaults to 0.2. Set AIGEN_TEMPERATURE
// to override.
func GetAIDocTemperature() float64 {
	v := envFloat("AIGEN_TEMPERATURE", 0.2)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// GetDiscoveryHintsCacheSize returns the LRU cache capacity for discovery chip
// hints keyed by sha256(idea). Defaults to 128.
func GetDiscoveryHintsCacheSize() int {
	v := envInt("DISCOVERY_HINTS_CACHE_SIZE", 128)
	if v < 1 {
		return 128
	}
	return v
}

// GetDiscoveryHintsTemperature returns the LLM temperature for discovery hint
// generation. Defaults to 0.0 for deterministic labels.
func GetDiscoveryHintsTemperature() float64 {
	v := envFloat("DISCOVERY_HINTS_TEMPERATURE", 0.0)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// GetArchEnricherEnabled returns whether the architecture enricher LLM pre-pass
// is enabled. Reads ARCH_ENRICHER_ENABLED; default true.
func GetArchEnricherEnabled() bool {
	v := os.Getenv("ARCH_ENRICHER_ENABLED")
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return b
}

// GetArchEnricherTimeoutSec returns the timeout in seconds for the architecture
// enricher LLM call. Reads ARCH_ENRICHER_TIMEOUT_SEC; default 30; clamped [5,120].
func GetArchEnricherTimeoutSec() int {
	v := envInt("ARCH_ENRICHER_TIMEOUT_SEC", 30)
	if v < 5 {
		return 5
	}
	if v > 120 {
		return 120
	}
	return v
}

// GetPlanEnricherEnabled returns whether the plan enricher LLM pre-pass is enabled.
// Reads PLAN_ENRICHER_ENABLED; default true.
func GetPlanEnricherEnabled() bool {
	v := os.Getenv("PLAN_ENRICHER_ENABLED")
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return b
}

// GetPlanEnricherTimeoutSec returns the timeout in seconds for the plan enricher
// LLM call. Reads PLAN_ENRICHER_TIMEOUT_SEC; default 45; clamped [10,120].
func GetPlanEnricherTimeoutSec() int {
	v := envInt("PLAN_ENRICHER_TIMEOUT_SEC", 45)
	if v < 10 {
		return 10
	}
	if v > 120 {
		return 120
	}
	return v
}

// GetReadmeEnricherEnabled returns whether the readme enricher LLM pre-pass is enabled.
// Reads README_ENRICHER_ENABLED; default true.
// Returns false when FINALIZE_MODE=deterministic regardless of README_ENRICHER_ENABLED.
func GetReadmeEnricherEnabled() bool {
	if GetFinalizeMode() == "deterministic" {
		return false
	}
	v := os.Getenv("README_ENRICHER_ENABLED")
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return b
}

// GetReadmeEnricherTimeoutSec returns the timeout in seconds for the readme enricher
// LLM call. Reads README_ENRICHER_TIMEOUT_SEC; default 45; clamped [10,120].
func GetReadmeEnricherTimeoutSec() int {
	v := envInt("README_ENRICHER_TIMEOUT_SEC", 45)
	if v < 10 {
		return 10
	}
	if v > 120 {
		return 120
	}
	return v
}

// ── Section-sequential AI document generation ───────────────────────────────

// GetAIGenSectionSequentialKeys returns document keys that use section-sequential
// enhancement instead of monolithic rewrite. Reads AIGEN_SECTION_SEQUENTIAL;
// default "architecture,plan".
func GetAIGenSectionSequentialKeys() []string {
	raw := envString("AIGEN_SECTION_SEQUENTIAL", "architecture,plan")
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"architecture", "plan"}
	}
	return out
}

// IsSectionSequentialDoc reports whether docKey uses the section-sequential path.
func IsSectionSequentialDoc(docKey string) bool {
	for _, k := range GetAIGenSectionSequentialKeys() {
		if k == docKey {
			return true
		}
	}
	return false
}

// GetAIGenCoherenceEnabled returns whether the post-merge coherence audit runs.
// Reads AIGEN_COHERENCE_ENABLED; default true.
func GetAIGenCoherenceEnabled() bool {
	v := os.Getenv("AIGEN_COHERENCE_ENABLED")
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return b
}

// GetAIGenCoherenceMinRatio returns the minimum retained section body ratio after
// coherence micro-fix. Reads AIGEN_COHERENCE_MIN_RATIO; default 0.95; clamped [0.5,1.0].
func GetAIGenCoherenceMinRatio() float64 {
	v := envFloat("AIGEN_COHERENCE_MIN_RATIO", 0.95)
	if v < 0.5 {
		return 0.5
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// GetAIGenCoherenceMaxEditRatio returns the maximum relative edit size for a
// coherence micro-fix. Reads AIGEN_COHERENCE_MAX_EDIT_RATIO; default 0.10;
// clamped [0.01,0.5].
func GetAIGenCoherenceMaxEditRatio() float64 {
	v := envFloat("AIGEN_COHERENCE_MAX_EDIT_RATIO", 0.10)
	if v < 0.01 {
		return 0.01
	}
	if v > 0.5 {
		return 0.5
	}
	return v
}

// GetAIGenPriorSectionMaxChars returns the character budget for prior-section
// context in section-sequential enhancement. Reads AIGEN_PRIOR_SECTION_MAX_CHARS;
// default 4000; clamped [500,20000].
func GetAIGenPriorSectionMaxChars() int {
	v := envInt("AIGEN_PRIOR_SECTION_MAX_CHARS", 4000)
	if v < 500 {
		return 500
	}
	if v > 20000 {
		return 20000
	}
	return v
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// envString reads an env var and returns defVal when absent or empty.
func envString(key, defVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defVal
}

// envInt reads an env var as an integer, returning defVal on parse failure or
// absence.
func envInt(key string, defVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defVal
	}
	return n
}

// envFloat reads an env var as a float64, returning defVal on parse failure or
// absence.
func envFloat(key string, defVal float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defVal
	}
	return f
}
