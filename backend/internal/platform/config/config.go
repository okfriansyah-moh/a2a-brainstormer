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
// in-flight LLM pipeline. Defaults to 5 minutes.
// Set ITERATION_TIMEOUT_SECONDS to override.
func GetIterationTimeout() time.Duration {
	return time.Duration(envInt("ITERATION_TIMEOUT_SECONDS", 300)) * time.Second
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
