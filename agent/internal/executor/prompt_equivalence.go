package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"a2a-brainstorm/agent/internal/llm"
)

// PromptParts separates instructions from authoritative state JSON.
type PromptParts struct {
	Instructions string
	StateJSON    string
}

// LegacyPromptParts extracts instruction and state content from a legacy LLMRequest.
func LegacyPromptParts(req llm.LLMRequest) PromptParts {
	return splitInstructionsAndState(req.SystemPrompt + "\n" + req.UserMessage)
}

// TieredPromptParts extracts instruction and state content from tiered blocks.
func TieredPromptParts(blocks []llm.PromptBlock) PromptParts {
	var b strings.Builder
	for _, block := range blocks {
		if block.Content != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(block.Content)
		}
	}
	return splitInstructionsAndState(b.String())
}

// splitInstructionsAndState separates the authoritative state JSON from instructions.
func splitInstructionsAndState(combined string) PromptParts {
	idx := strings.Index(combined, stateSectionPrefix)
	if idx < 0 {
		return PromptParts{Instructions: NormalizeInstructions(combined), StateJSON: "{}"}
	}
	instructions := combined[:idx]
	stateTail := combined[idx+len(stateSectionPrefix):]
	stateJSON := extractStateJSONFromTail(stateTail)
	return PromptParts{
		Instructions: NormalizeInstructions(instructions),
		StateJSON:    NormalizeStateJSON(stateJSON),
	}
}

func extractStateJSONFromTail(tail string) string {
	end := strings.Index(tail, "\n\nReturn your role-scoped JSON delta now.")
	if end < 0 {
		return strings.TrimSpace(tail)
	}
	return strings.TrimSpace(tail[:end])
}

// NormalizeInstructions collapses whitespace for stable comparison.
func NormalizeInstructions(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// NormalizeStateJSON canonicalizes JSON for byte-stable comparison.
func NormalizeStateJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return string(out)
}

// AssertSemanticEquivalent verifies tiered assembly preserves legacy instructions and state.
func AssertSemanticEquivalent(legacy llm.LLMRequest, tieredBlocks []llm.PromptBlock) error {
	leg := LegacyPromptParts(legacy)
	tierInstr := tieredInstructionParts(tieredBlocks)

	if leg.StateJSON != extractStateFromTieredBlocks(tieredBlocks) {
		return fmt.Errorf("state JSON mismatch")
	}

	if leg.Instructions == tierInstr {
		return nil
	}
	if containsNormalized(tierInstr, leg.Instructions) {
		return nil
	}
	return fmt.Errorf("tiered instructions missing legacy content")
}

func tieredInstructionParts(blocks []llm.PromptBlock) string {
	var b strings.Builder
	for _, block := range blocks {
		if strings.HasPrefix(block.Content, "SESSION_ANCHOR:") {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(block.Content)
	}
	return splitInstructionsAndState(b.String()).Instructions
}

func extractStateFromTieredBlocks(blocks []llm.PromptBlock) string {
	var b strings.Builder
	for _, block := range blocks {
		if strings.HasPrefix(block.Content, "SESSION_ANCHOR:") {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(block.Content)
	}
	return splitInstructionsAndState(b.String()).StateJSON
}

func containsNormalized(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(haystack, needle)
}

// CanonicalRequestHash returns a stable hash for retry-only response caching.
func CanonicalRequestHash(req llm.LLMRequest) string {
	h := sha256.New()
	writeHashField(h, "system_prompt", req.SystemPrompt)
	writeHashField(h, "user_message", req.UserMessage)
	writeHashField(h, "temperature", strconv.FormatFloat(req.Temperature, 'f', 4, 64))
	if req.Tiered == nil {
		writeHashField(h, "tiered_present", "0")
	} else {
		writeHashField(h, "tiered_present", "1")
		writeHashField(h, "tiered_session_id", req.Tiered.SessionID)
		writeHashField(h, "tiered_agent_id", req.Tiered.AgentID)
		writeHashField(h, "tiered_provider", req.Tiered.Provider)
		writeHashField(h, "tiered_model", req.Tiered.Model)
		writeHashField(h, "tiered_block_count", strconv.Itoa(len(req.Tiered.Blocks)))
		for i, b := range req.Tiered.Blocks {
			prefix := fmt.Sprintf("tiered_block_%d", i)
			writeHashField(h, prefix+"_role", b.Role)
			writeHashField(h, prefix+"_content", b.Content)
			writeHashField(h, prefix+"_cache_policy", strconv.Itoa(int(b.CachePolicy)))
		}
		writeHashField(h, "tiered_message_count", strconv.Itoa(len(req.Tiered.Messages)))
		for i, m := range req.Tiered.Messages {
			prefix := fmt.Sprintf("tiered_message_%d", i)
			writeHashField(h, prefix+"_role", m.Role)
			writeHashField(h, prefix+"_content", m.Content)
			writeHashField(h, prefix+"_cache_policy", strconv.Itoa(int(m.CachePolicy)))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeHashField(h interface{ Write([]byte) (int, error) }, key, value string) {
	_, _ = h.Write([]byte(key))
	_, _ = h.Write([]byte{'='})
	_, _ = h.Write([]byte(strconv.Itoa(len(value))))
	_, _ = h.Write([]byte{':'})
	_, _ = h.Write([]byte(value))
	_, _ = h.Write([]byte{'|'})
}
