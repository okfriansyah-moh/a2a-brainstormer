package iteration

import (
	"strings"
	"time"
)

const (
	tokenBatchMinChars = 96
	tokenBatchMaxWait  = 80 * time.Millisecond
)

// agentTokenBatcher coalesces high-frequency LLM tokens before SSE emission so
// the broadcaster's small per-subscriber buffer is not overwhelmed.
type agentTokenBatcher struct {
	buf       strings.Builder
	lastFlush time.Time
}

func newAgentTokenBatcher() *agentTokenBatcher {
	return &agentTokenBatcher{lastFlush: time.Now()}
}

func (b *agentTokenBatcher) append(token string) string {
	if token == "" {
		return ""
	}
	b.buf.WriteString(token)
	now := time.Now()
	if b.buf.Len() >= tokenBatchMinChars || now.Sub(b.lastFlush) >= tokenBatchMaxWait {
		return b.flush()
	}
	return ""
}

func (b *agentTokenBatcher) flush() string {
	if b.buf.Len() == 0 {
		return ""
	}
	out := b.buf.String()
	b.buf.Reset()
	b.lastFlush = time.Now()
	return out
}
