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
