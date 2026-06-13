package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/platform/llm"
)

func TestNewGlobalLLMProvider_UsesOpenCodeWhenConfigured(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "opencode")
	t.Setenv("GLOBAL_OPENCODE_BASE_URL", "http://opencode:4096")
	t.Setenv("GLOBAL_OPENCODE_MODEL", "github-copilot/claude-sonnet-4.6")
	t.Setenv("OPENCODE_USERNAME_REF", "TEST_OPENCODE_USERNAME")
	t.Setenv("OPENCODE_PASSWORD_REF", "TEST_OPENCODE_PASSWORD")
	t.Setenv("TEST_OPENCODE_USERNAME", "opencode")
	t.Setenv("TEST_OPENCODE_PASSWORD", "secret")

	provider := newGlobalLLMProvider()
	if _, ok := provider.(*llm.OpenCodeProvider); !ok {
		t.Fatalf("newGlobalLLMProvider() type = %T, want *llm.OpenCodeProvider", provider)
	}
}

func TestNewGlobalLLMProvider_FallsBackToCopilot(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "copilot")
	t.Setenv("GLOBAL_LLM_MODEL", "gpt-4o")
	t.Setenv("GLOBAL_LLM_CREDENTIAL_REF", "TEST_COPILOT_KEY")
	t.Setenv("TEST_COPILOT_KEY", "test-key")

	provider := newGlobalLLMProvider()
	if _, ok := provider.(*llm.CopilotProvider); !ok {
		t.Fatalf("newGlobalLLMProvider() type = %T, want *llm.CopilotProvider", provider)
	}
}

func TestHasDiscoveryHintsCredentials_OpenCodeMissingUsername(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "opencode")
	t.Setenv("OPENCODE_USERNAME_REF", "TEST_OPENCODE_USERNAME")
	t.Setenv("OPENCODE_PASSWORD_REF", "TEST_OPENCODE_PASSWORD")
	// Intentionally omit TEST_OPENCODE_USERNAME.
	t.Setenv("TEST_OPENCODE_PASSWORD", "secret")

	if got := hasDiscoveryHintsCredentials("opencode"); got {
		t.Fatalf("hasDiscoveryHintsCredentials(opencode) = %v, want false when username is missing", got)
	}
}

func TestHasDiscoveryHintsCredentials_OpenCodePresent(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "opencode")
	t.Setenv("OPENCODE_USERNAME_REF", "TEST_OPENCODE_USERNAME")
	t.Setenv("OPENCODE_PASSWORD_REF", "TEST_OPENCODE_PASSWORD")
	t.Setenv("TEST_OPENCODE_USERNAME", "opencode")
	t.Setenv("TEST_OPENCODE_PASSWORD", "secret")

	if got := hasDiscoveryHintsCredentials("opencode"); !got {
		t.Fatalf("hasDiscoveryHintsCredentials(opencode) = %v, want true when both credentials are present", got)
	}
}

func TestHasDiscoveryHintsCredentials_CopilotMissingCredential(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "copilot")
	t.Setenv("GLOBAL_LLM_CREDENTIAL_REF", "TEST_COPILOT_KEY")
	// Intentionally omit TEST_COPILOT_KEY.

	if got := hasDiscoveryHintsCredentials("copilot"); got {
		t.Fatalf("hasDiscoveryHintsCredentials(copilot) = %v, want false when credential is missing", got)
	}
}

func TestGlobalLLMConfigHandler_PutUpdatesEnvVars(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "deepseek")
	t.Setenv("GLOBAL_LLM_MODEL", "deepseek-v4-flash")
	t.Setenv("GLOBAL_LLM_CREDENTIAL_REF", "DEEPSEEK_API_KEY")

	handler := globalLLMConfigHandler()
	req := httptest.NewRequest(http.MethodPut, "/api/config/global-llm", strings.NewReader(`{"provider":"openai","model":"gpt-5.4","credential_ref":"OPENAI_API_KEY"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got struct {
		Provider  string `json:"provider"`
		Model     string `json:"model"`
		Available bool   `json:"available"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Provider != "openai" || got.Model != "gpt-5.4" {
		t.Fatalf("Decode() = %+v, want provider=openai model=gpt-5.4", got)
	}
	if gotEnv := os.Getenv("GLOBAL_LLM_PROVIDER"); gotEnv != "openai" {
		t.Fatalf("GLOBAL_LLM_PROVIDER = %q, want %q", gotEnv, "openai")
	}
	if gotCred := os.Getenv("GLOBAL_LLM_CREDENTIAL_REF"); gotCred != "OPENAI_API_KEY" {
		t.Fatalf("GLOBAL_LLM_CREDENTIAL_REF = %q, want %q", gotCred, "OPENAI_API_KEY")
	}
}

func TestGlobalLLMConfigHandler_PutPreservesCredentialRefWhenOmitted(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "deepseek")
	t.Setenv("GLOBAL_LLM_MODEL", "deepseek-v4-flash")
	t.Setenv("GLOBAL_LLM_CREDENTIAL_REF", "DEEPSEEK_API_KEY")

	handler := globalLLMConfigHandler()
	req := httptest.NewRequest(http.MethodPut, "/api/config/global-llm", strings.NewReader(`{"provider":"openai","model":"gpt-5.4"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotCred := os.Getenv("GLOBAL_LLM_CREDENTIAL_REF"); gotCred != "DEEPSEEK_API_KEY" {
		t.Fatalf("GLOBAL_LLM_CREDENTIAL_REF = %q, want %q", gotCred, "DEEPSEEK_API_KEY")
	}
}

func TestGlobalLLMConfigHandler_PutDefaultsToDeepSeek(t *testing.T) {
	t.Setenv("GLOBAL_LLM_PROVIDER", "")
	t.Setenv("GLOBAL_LLM_MODEL", "")
	t.Setenv("GLOBAL_LLM_CREDENTIAL_REF", "")

	handler := globalLLMConfigHandler()
	req := httptest.NewRequest(http.MethodPut, "/api/config/global-llm", strings.NewReader(`{"provider":"","model":"","credential_ref":""}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got struct {
		Provider  string `json:"provider"`
		Model     string `json:"model"`
		Available bool   `json:"available"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Provider != "deepseek" || got.Model != "deepseek-v4-flash" {
		t.Fatalf("Decode() = %+v, want provider=deepseek model=deepseek-v4-flash", got)
	}
	if gotCred := os.Getenv("GLOBAL_LLM_CREDENTIAL_REF"); gotCred != "DEEPSEEK_API_KEY" {
		t.Fatalf("GLOBAL_LLM_CREDENTIAL_REF = %q, want %q", gotCred, "DEEPSEEK_API_KEY")
	}
}
