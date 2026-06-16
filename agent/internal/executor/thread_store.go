package executor

import (
	"sync"

	"a2a-brainstorm/agent/internal/config"
	"a2a-brainstorm/agent/internal/llm"
)

// threadKey identifies one agent's prompt thread within a session.
type threadKey struct {
	sessionID string
	agentID   string
}

// ThreadStore holds multi-turn message threads with state replacement (Option B).
type ThreadStore struct {
	mu      sync.Mutex
	threads map[threadKey][]llm.PromptBlock
	max     int
	order   []threadKey
}

// NewThreadStore creates a thread store with the given LRU capacity.
func NewThreadStore(max int) *ThreadStore {
	if max < 1 {
		max = 256
	}
	return &ThreadStore{
		threads: make(map[threadKey][]llm.PromptBlock),
		max:     max,
	}
}

var defaultThreadStore = NewThreadStore(config.GetPromptCacheThreadMax())

// MessagesFor returns the thread messages for payload, replacing the state turn.
func (ts *ThreadStore) MessagesFor(payload BrainstormPayload) []llm.PromptBlock {
	key := threadKey{sessionID: payload.SessionID, agentID: payload.AgentID}
	stateJSON := marshalStateJSON(payload)

	tier1 := buildInjectedSystemPrompt(payload) + "\n\n" +
		deltaOutputPreamble + "\n" + roleDeltaInstruction(payload.Role)
	anchor := buildSessionAnchor(payload.OutputDocs)
	stateContent := buildThreadStateContent(payload, stateJSON)

	ts.mu.Lock()
	defer ts.mu.Unlock()

	msgs, ok := ts.threads[key]
	if !ok {
		ts.evictIfNeeded()
		msgs = []llm.PromptBlock{
			{Role: "system", Content: tier1, CachePolicy: llm.CacheEphemeral},
			{Role: "user", Content: anchor, CachePolicy: llm.CacheEphemeral},
		}
		ts.threads[key] = msgs
		ts.order = append(ts.order, key)
	}

	msgs = ts.replaceStateMessage(msgs, stateContent)
	ts.threads[key] = msgs
	ts.touch(key)
	return cloneBlocks(msgs)
}

func (ts *ThreadStore) replaceStateMessage(msgs []llm.PromptBlock, stateContent string) []llm.PromptBlock {
	out := make([]llm.PromptBlock, 0, len(msgs)+1)
	for _, m := range msgs {
		if m.Role == "user" && stringsHasPrefix(m.Content, currentStateLabel+":") {
			continue
		}
		if m.Role == "user" && stringsHasPrefix(m.Content, "CURRENT_STATE") {
			continue
		}
		out = append(out, m)
	}
	out = append(out, llm.PromptBlock{
		Role:        "user",
		Content:     stateContent,
		CachePolicy: llm.CacheNone,
	})
	return out
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func (ts *ThreadStore) touch(key threadKey) {
	for i, k := range ts.order {
		if k == key {
			ts.order = append(ts.order[:i], ts.order[i+1:]...)
			break
		}
	}
	ts.order = append(ts.order, key)
}

func (ts *ThreadStore) evictIfNeeded() {
	for len(ts.order) >= ts.max && len(ts.order) > 0 {
		oldest := ts.order[0]
		ts.order = ts.order[1:]
		delete(ts.threads, oldest)
	}
}

func cloneBlocks(in []llm.PromptBlock) []llm.PromptBlock {
	out := make([]llm.PromptBlock, len(in))
	copy(out, in)
	return out
}

// Reset clears all threads (for tests).
func (ts *ThreadStore) Reset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.threads = make(map[threadKey][]llm.PromptBlock)
	ts.order = nil
}
