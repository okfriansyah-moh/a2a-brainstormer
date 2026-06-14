package state

import (
	"fmt"
	"strings"
)

// EnrichIdeaFromSession backfills idea fields used for document titles and
// prompts when pipeline merges replaced the idea map with a sparse agent delta
// that dropped the original user text.
func EnrichIdeaFromSession(s CanonicalState, sessionIdea string) CanonicalState {
	sessionIdea = strings.TrimSpace(sessionIdea)
	idea := cloneIdeaMap(s.Idea)

	text := ideaString(idea, "text")
	if text == "" && sessionIdea != "" {
		idea["text"] = sessionIdea
		text = sessionIdea
	}

	if ideaString(idea, "name") == "" && ideaString(idea, "title") == "" && text != "" {
		if title := SummarizeIdeaTitle(text); title != "" {
			idea["title"] = title
		}
	}

	if ideaString(idea, "summary") == "" && text != "" {
		idea["summary"] = summarizeIdeaSummary(text)
	}

	s.Idea = idea
	return s
}

func cloneIdeaMap(m map[string]any) map[string]any {
	if len(m) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func ideaString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// SummarizeIdeaTitle derives a short product name from free-form idea text.
func SummarizeIdeaTitle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	prefixes := []string{
		"i have an idea where i want to build ",
		"i have an idea to build ",
		"i want to build ",
		"i want to create ",
		"build an ",
		"build a ",
		"create an ",
		"create a ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			raw = strings.TrimSpace(raw[len(p):])
			break
		}
	}
	raw = firstSentence(raw, 120)
	raw = strings.TrimRight(raw, ".,;:!?")
	return truncateAtWord(raw, 60)
}

func summarizeIdeaSummary(text string) string {
	joined := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	runes := []rune(joined)
	if len(runes) <= 150 {
		return joined
	}
	return string(runes[:150]) + "…"
}

func firstSentence(s string, max int) string {
	cut := strings.IndexAny(s, ".!?\n")
	if cut > 0 {
		s = strings.TrimSpace(s[:cut])
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func truncateAtWord(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	cut := max
	for cut > 0 && runes[cut] != ' ' && runes[cut] != '\t' {
		cut--
	}
	if cut == 0 {
		cut = max
	}
	return strings.TrimRight(string(runes[:cut]), " \t,;:-")
}
