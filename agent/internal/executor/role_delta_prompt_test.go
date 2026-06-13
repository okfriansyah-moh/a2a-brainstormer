package executor

import (
	"strings"
	"testing"
)

func TestRoleDeltaInstruction_BuildRole(t *testing.T) {
	msg := roleDeltaInstruction("build")
	for _, want := range []string{"architecture", "execution_plan", "ONLY"} {
		if !strings.Contains(msg, want) {
			t.Errorf("instruction missing %q", want)
		}
	}
}

func TestRoleDeltaInstruction_ReviewRole(t *testing.T) {
	msg := roleDeltaInstruction("review")
	for _, want := range []string{"risks", "open_questions"} {
		if !strings.Contains(msg, want) {
			t.Errorf("instruction missing %q", want)
		}
	}
}
