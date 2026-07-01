package state

import (
	"regexp"
	"strings"
)

const intentJaccardThreshold = 0.72

// topicJaccardThreshold is used for assumptions and open questions where the same
// topic may be restated with extra detail or a resolution note.
const topicJaccardThreshold = 0.50

const minSharedTopicTokens = 3

var listPrefixPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(phase|task|step|milestone)\s*\d+\s*[-–—:.)]+\s*`),
	regexp.MustCompile(`(?i)^(assumption|question|open question|q)\s*\d+\s*[-–—:.)]+\s*`),
	regexp.MustCompile(`^\d+\s*[-–—:.)]+\s*`),
}

var stopwords = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"be": {}, "been": {}, "being": {}, "have": {}, "has": {}, "had": {},
	"do": {}, "does": {}, "did": {}, "will": {}, "would": {}, "could": {},
	"should": {}, "may": {}, "might": {}, "must": {}, "shall": {},
	"to": {}, "of": {}, "in": {}, "for": {}, "on": {}, "with": {}, "at": {},
	"by": {}, "from": {}, "as": {}, "into": {}, "through": {}, "during": {},
	"before": {}, "after": {}, "above": {}, "below": {}, "between": {},
	"and": {}, "or": {}, "but": {}, "if": {}, "then": {}, "than": {},
	"that": {}, "this": {}, "these": {}, "those": {}, "it": {}, "its": {},
	"they": {}, "them": {}, "their": {}, "we": {}, "our": {}, "you": {}, "your": {},
	"not": {}, "no": {}, "nor": {}, "so": {}, "such": {}, "only": {}, "own": {},
	"same": {}, "other": {}, "some": {}, "any": {}, "each": {}, "all": {},
	"can": {}, "just": {}, "also": {}, "now": {}, "new": {}, "via": {},
}

// stripListPrefix removes leading enumeration labels such as "Phase 1 —" or "Task 3:".
func stripListPrefix(s string) string {
	out := strings.TrimSpace(s)
	for {
		prev := out
		for _, re := range listPrefixPatterns {
			out = strings.TrimSpace(re.ReplaceAllString(out, ""))
		}
		if out == prev {
			break
		}
	}
	return out
}

func significantTokens(s string) map[string]struct{} {
	tokens := make(map[string]struct{})
	for _, raw := range strings.Fields(normaliseText(s)) {
		tok := strings.Trim(raw, ".,;:!?\"'()[]{}")
		if len(tok) < 3 {
			continue
		}
		if _, skip := stopwords[tok]; skip {
			continue
		}
		tokens[tok] = struct{}{}
	}
	return tokens
}

func tokenJaccard(a, b string) float64 {
	ta := significantTokens(a)
	tb := significantTokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for tok := range ta {
		if _, ok := tb[tok]; ok {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func sharedSignificantTokenCount(a, b string) int {
	ta := significantTokens(a)
	tb := significantTokens(b)
	n := 0
	for tok := range ta {
		if _, ok := tb[tok]; ok {
			n++
		}
	}
	return n
}

func isContainedIntent(shorter, longer string) bool {
	if shorter == "" || longer == "" {
		return false
	}
	if strings.Contains(longer, shorter) {
		ratio := float64(len(shorter)) / float64(len(longer))
		return ratio >= 0.55
	}
	return false
}

func stripParentheticals(s string) string {
	re := regexp.MustCompile(`\([^)]*\)`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}

func promptStem(s string) string {
	cleaned := stripParentheticals(s)
	if idx := strings.Index(cleaned, "?"); idx >= 0 {
		cleaned = cleaned[:idx]
	}
	return normaliseText(firstNWords(cleaned, 14))
}

func supersedesPrior(a, b string) bool {
	shared := sharedSignificantTokenCount(a, b)
	if shared < 3 {
		return false
	}
	markers := []string{
		"resolving the previous",
		"resolves the previous",
		"now separated",
		"now assigned",
		"supersedes",
		"resolved:",
	}
	for _, text := range []string{strings.ToLower(a), strings.ToLower(b)} {
		for _, m := range markers {
			if strings.Contains(text, m) {
				return true
			}
		}
	}
	return false
}

// sameIntent reports whether two free-text items express the same underlying point.
func sameIntent(a, b string) bool {
	na := normaliseText(a)
	nb := normaliseText(b)
	if na == "" || nb == "" {
		return na == nb && na != ""
	}
	if na == nb {
		return true
	}

	ca := normaliseText(stripListPrefix(a))
	cb := normaliseText(stripListPrefix(b))
	if ca != "" && ca == cb {
		return true
	}

	stemA := promptStem(a)
	stemB := promptStem(b)
	if stemA != "" && stemB != "" {
		if stemA == stemB {
			return true
		}
		if isContainedIntent(stemA, stemB) || isContainedIntent(stemB, stemA) {
			return true
		}
		if tokenJaccard(stemA, stemB) >= 0.82 {
			return true
		}
	}

	short, long := na, nb
	if len(short) > len(long) {
		short, long = long, short
	}
	if isContainedIntent(short, long) {
		return true
	}

	if tokenJaccard(na, nb) >= intentJaccardThreshold {
		return true
	}

	shared := sharedSignificantTokenCount(na, nb)
	if shared >= minSharedTopicTokens && tokenJaccard(na, nb) >= topicJaccardThreshold {
		return true
	}
	if shared >= 4 && tokenJaccard(na, nb) >= 0.35 {
		return true
	}
	if supersedesPrior(a, b) {
		return true
	}
	return false
}

// preferText keeps the more informative variant of two intent-equivalent strings.
func preferText(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if len(b) > len(a) {
		return b
	}
	if len(a) > len(b) {
		return a
	}
	lowerA := strings.ToLower(a)
	lowerB := strings.ToLower(b)
	for _, m := range []string{"resolved:", "resolving the previous", "now separated", "now assigned"} {
		if strings.Contains(lowerB, m) && !strings.Contains(lowerA, m) {
			return b
		}
		if strings.Contains(lowerA, m) && !strings.Contains(lowerB, m) {
			return a
		}
	}
	return a
}

func mergeIntentStrings(base, incoming []string) []string {
	var out []string

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for i, existing := range out {
			if sameIntent(existing, s) {
				out[i] = preferText(existing, s)
				return
			}
		}
		out = append(out, s)
	}

	for _, s := range base {
		add(s)
	}
	for _, s := range incoming {
		add(s)
	}
	return out
}

func stepIntentKey(s Step) string {
	title := stripListPrefix(s.Title)
	if title != "" {
		return normaliseText(title)
	}
	if s.Description != "" {
		return normaliseText(firstNWords(s.Description, 12))
	}
	return ""
}

func firstNWords(s string, n int) string {
	fields := strings.Fields(s)
	if len(fields) <= n {
		return strings.Join(fields, " ")
	}
	return strings.Join(fields[:n], " ")
}

func stepsSameIntent(a, b Step) bool {
	ka := stepIntentKey(a)
	kb := stepIntentKey(b)
	if ka != "" && ka == kb {
		return true
	}

	ta := normaliseText(stripListPrefix(a.Title))
	tb := normaliseText(stripListPrefix(b.Title))
	if ta != "" && ta == tb {
		return true
	}

	da := normaliseText(a.Description)
	db := normaliseText(b.Description)
	if da == "" || db == "" {
		return false
	}
	if isContainedIntent(da, db) || isContainedIntent(db, da) {
		return true
	}
	return tokenJaccard(da, db) >= 0.68
}

func hasEnumeratedPrefix(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, prefix := range []string{"phase ", "task ", "step ", "milestone "} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func preferStepTitle(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	aEnum := hasEnumeratedPrefix(a)
	bEnum := hasEnumeratedPrefix(b)
	switch {
	case aEnum && !bEnum:
		return b
	case bEnum && !aEnum:
		return a
	default:
		if len(stripListPrefix(b)) > len(stripListPrefix(a)) {
			return b
		}
		return a
	}
}

// Compact removes intent-equivalent duplicates within a single canonical state
// snapshot. It is used before dispatch so agents do not re-process repeated
// execution plan steps, assumptions, or open questions carried over from prior
// passes.
func Compact(cs CanonicalState) CanonicalState {
	cs.ExecutionPlan = mergeSteps(cs.ExecutionPlan, nil)
	cs.Risks = mergeRisks(cs.Risks, nil)
	cs.Assumptions = mergeIntentStrings(cs.Assumptions, nil)
	cs.OpenQuestions = mergeIntentStrings(cs.OpenQuestions, nil)
	return cs
}

func mergePreferDetailedStep(a, b Step) Step {
	out := a
	if wordCount(b.Description) > wordCount(a.Description) {
		out.Description = b.Description
	} else if wordCount(a.Description) == 0 && b.Description != "" {
		out.Description = b.Description
	}
	out.Title = preferStepTitle(a.Title, b.Title)

	if out.Objective == "" && b.Objective != "" {
		out.Objective = b.Objective
	}
	if len(out.Deliverables) == 0 && len(b.Deliverables) > 0 {
		out.Deliverables = append([]string(nil), b.Deliverables...)
	}
	if out.Scope == "" && b.Scope != "" {
		out.Scope = b.Scope
	}
	if out.FailureHandling == "" && b.FailureHandling != "" {
		out.FailureHandling = b.FailureHandling
	}
	if len(out.ExitCriteria) == 0 && len(b.ExitCriteria) > 0 {
		out.ExitCriteria = append([]string(nil), b.ExitCriteria...)
	}
	if len(out.BlockingDependencies) == 0 && len(b.BlockingDependencies) > 0 {
		out.BlockingDependencies = append([]string(nil), b.BlockingDependencies...)
	}
	if len(out.FunctionContracts) == 0 && len(b.FunctionContracts) > 0 {
		out.FunctionContracts = append([]string(nil), b.FunctionContracts...)
	}
	return out
}
