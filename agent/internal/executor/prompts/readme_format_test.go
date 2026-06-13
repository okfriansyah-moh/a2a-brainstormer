// Package prompts — tests for InjectIfReadmeOutput.
package prompts

import (
	"strings"
	"testing"
)

// TestInjectIfReadmeOutput_InjectsWhenPresent verifies that ReadmeFormat is
// appended to base when "readme" is present in outputDocs.
func TestInjectIfReadmeOutput_InjectsWhenPresent(t *testing.T) {
	base := "system prompt base"
	result := InjectIfReadmeOutput(base, []string{"readme"})

	if !strings.HasPrefix(result, base) {
		t.Errorf("result does not start with base prompt")
	}
	if !strings.Contains(result, ReadmeFormat) {
		t.Errorf("result does not contain ReadmeFormat")
	}
	if result == base {
		t.Errorf("result is identical to base; expected ReadmeFormat to be appended")
	}
}

// TestInjectIfReadmeOutput_NoopWhenAbsent verifies that base is returned
// unchanged when "readme" is not in outputDocs.
func TestInjectIfReadmeOutput_NoopWhenAbsent(t *testing.T) {
	base := "system prompt base"
	result := InjectIfReadmeOutput(base, []string{"plan"})

	if result != base {
		t.Errorf("result changed when 'readme' absent: got %q; want %q", result, base)
	}
}

// TestInjectIfReadmeOutput_CaseInsensitive verifies that "Readme", "README",
// and " readme " all trigger injection, matching the plan helper's behaviour.
func TestInjectIfReadmeOutput_CaseInsensitive(t *testing.T) {
	base := "system prompt base"
	for _, doc := range []string{"README", "Readme", " readme "} {
		result := InjectIfReadmeOutput(base, []string{doc})
		if result == base {
			t.Errorf("expected injection for doc=%q but got base unchanged", doc)
		}
	}
}

// TestReadmeFormat_Length verifies ReadmeFormat stays within the ≤3000 char
// budget documented on the constant. Prompt bloat raises per-call token cost.
func TestReadmeFormat_Length(t *testing.T) {
	const maxLen = 3000
	if got := len(ReadmeFormat); got > maxLen {
		t.Errorf("ReadmeFormat exceeds %d chars: got %d — trim the prompt", maxLen, got)
	}
}
