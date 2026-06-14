package iteration

import "testing"

func TestAgentTokenBatcher_CoalescesAndFlushes(t *testing.T) {
	b := newAgentTokenBatcher()

	if got := b.append("abc"); got != "" {
		t.Fatalf("first append = %q, want empty", got)
	}
	if got := b.flush(); got != "abc" {
		t.Fatalf("flush() = %q, want abc", got)
	}
	if got := b.flush(); got != "" {
		t.Fatalf("second flush() = %q, want empty", got)
	}
}

func TestAgentTokenBatcher_FlushesOnSizeThreshold(t *testing.T) {
	b := newAgentTokenBatcher()
	var out string
	for i := 0; i < tokenBatchMinChars; i++ {
		out = b.append("x")
	}
	if out == "" {
		t.Fatal("expected flush once min chars reached")
	}
}
