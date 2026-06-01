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
// reached before the engine is allowed to converge. Defaults to 0.90 — the
// pipeline will not stop until at least one agent reports 90% confidence.
// Set MIN_CONFIDENCE_FLOOR to override.
func GetMinConfidenceFloor() float64 {
	return envFloat("MIN_CONFIDENCE_FLOOR", 0.90)
}

// ── Global LLM defaults ───────────────────────────────────────────────────────

// GetGlobalLLMProvider returns the default LLM provider name.
// Allowed values: "copilot" | "claude". Defaults to "copilot".
func GetGlobalLLMProvider() string {
	return envString("GLOBAL_LLM_PROVIDER", "copilot")
}

// GetGlobalLLMModel returns the default LLM model name. Defaults to "gpt-4o".
func GetGlobalLLMModel() string {
	return envString("GLOBAL_LLM_MODEL", "gpt-4o")
}

// GetGlobalLLMCredentialRef returns the env var NAME that holds the global LLM
// API key. This is a reference (env var name), never the raw key value.
// Defaults to "COPILOT_API_KEY".
func GetGlobalLLMCredentialRef() string {
	return envString("GLOBAL_LLM_CREDENTIAL_REF", "COPILOT_API_KEY")
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

// GetAgentCallTimeout returns the HTTP timeout for a single A2A agent call.
// LLM inference (especially Claude on large prompts) can take several minutes.
// Defaults to 10 minutes. Set AGENT_CALL_TIMEOUT_SECONDS to override.
func GetAgentCallTimeout() time.Duration {
	return time.Duration(envInt("AGENT_CALL_TIMEOUT_SECONDS", 600)) * time.Second
}

// GetFinalizeTimeout returns the maximum duration allowed for one finalize
// document-generation call (deterministic scaffold + optional AI enhance pass).
// This timeout is applied per-request, independently of the server WriteTimeout.
// Defaults to 10 minutes. Set FINALIZE_TIMEOUT_SECONDS to override.
func GetFinalizeTimeout() time.Duration {
	return time.Duration(envInt("FINALIZE_TIMEOUT_SECONDS", 600)) * time.Second
}

// GetFinalizeMode returns the document-generation strategy for session
// finalize. Valid values: "deterministic", "hybrid", "ai". Defaults to
// "hybrid". Unknown values fall back to "deterministic" downstream.
// Set FINALIZE_MODE to override.
func GetFinalizeMode() string {
	return envString("FINALIZE_MODE", "hybrid")
}

// GetSkillBundlePaths returns the comma-separated list of skill file paths
// (relative to the repository root) loaded into the AI doc-generator system
// prompt. The default set covers the five canonical skills used for output
// documents. Set SKILL_BUNDLE_PATHS to override (comma-separated).
func GetSkillBundlePaths() []string {
	raw := envString("SKILL_BUNDLE_PATHS", strings.Join(defaultSkillBundlePaths, ","))
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// defaultSkillBundlePaths is the canonical ordered list of skills injected as
// system-prompt context for AI document generation. See docs/PLAN.md §8.27.
var defaultSkillBundlePaths = []string{
	".github/skills/modularity/SKILL.md",
	".github/skills/vertical-slice/SKILL.md",
	".github/skills/api-design/SKILL.md",
	".github/skills/roadmap-spec/SKILL.md",
	".github/skills/plan-management/SKILL.md",
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
