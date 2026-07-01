package iteration

import (
	"testing"
	"time"
)

func TestStreamPhaseThrottler_firstEmitAlways(t *testing.T) {
	t.Parallel()
	th := &streamPhaseThrottler{}
	if !th.shouldEmit(96) {
		t.Fatal("expected first emit at 96 chars")
	}
	th.markEmitted(96)
}

func TestStreamPhaseThrottler_blocksRapidSmallDeltas(t *testing.T) {
	t.Parallel()
	th := &streamPhaseThrottler{}
	th.markEmitted(96)
	if th.shouldEmit(150) {
		t.Fatal("expected throttle before interval and char delta")
	}
}

func TestStreamPhaseThrottler_emitsOnCharDelta(t *testing.T) {
	t.Parallel()
	th := &streamPhaseThrottler{}
	th.markEmitted(96)
	if !th.shouldEmit(96+streamPhaseMinCharDelta) {
		t.Fatal("expected emit after char delta threshold")
	}
}

func TestStreamPhaseThrottler_emitsOnInterval(t *testing.T) {
	t.Parallel()
	th := &streamPhaseThrottler{lastEmit: time.Now().Add(-streamPhaseMinInterval)}
	th.lastChars = 200
	if !th.shouldEmit(250) {
		t.Fatal("expected emit after interval elapsed")
	}
}
