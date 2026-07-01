package iteration

import "time"

const (
	streamPhaseMinInterval  = 500 * time.Millisecond
	streamPhaseMinCharDelta = 400
)

// streamPhaseThrottler limits high-frequency agent.phase SSE events during
// token streaming so the broadcaster ring buffer is not flooded and slow
// subscribers are not dropped mid-pass.
type streamPhaseThrottler struct {
	lastEmit  time.Time
	lastChars int
}

func (t *streamPhaseThrottler) shouldEmit(streamChars int) bool {
	if streamChars <= 0 {
		return false
	}
	if t.lastChars == 0 {
		return true
	}
	if streamChars-t.lastChars >= streamPhaseMinCharDelta {
		return true
	}
	return time.Since(t.lastEmit) >= streamPhaseMinInterval
}

func (t *streamPhaseThrottler) markEmitted(streamChars int) {
	t.lastEmit = time.Now()
	t.lastChars = streamChars
}
