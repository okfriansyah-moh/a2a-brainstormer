package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func hashString(s string) uint32 {
	var h uint32
	for _, c := range []byte(s) {
		h = h*31 + uint32(c)
	}
	return h
}

// CanonicalRequestHash returns a stable hash for retry-only response caching.
func CanonicalRequestHash(req llm.LLMRequest) string {
	var buf bytes.Buffer
	buf.WriteString(req.SystemPrompt)
	buf.WriteByte(0)
	buf.WriteString(req.UserMessage)
	buf.WriteByte(0)
	buf.WriteString(fmt.Sprintf("%.4f", req.Temperature))
	if req.Tiered != nil {
		for _, b := range req.Tiered.Blocks {
			buf.WriteString(b.Role)
			buf.WriteString(b.Content)
		}
		for _, m := range req.Tiered.Messages {
			buf.WriteString(m.Role)
			buf.WriteString(m.Content)
		}
		buf.WriteString(req.Tiered.Provider)
		buf.WriteString(req.Tiered.Model)
	}
	return fmt.Sprintf("%x", hashString(buf.String()))
}
