// Package iteration — preview.go provides an in-memory store for per-agent
// preview results produced by RunSingleAgent.
//
// Previews are ephemeral: they live only in process memory and are cleared on
// server restart. The store is safe for concurrent access via its internal
// RWMutex.
//
// Key structure: sessionID → agentID → PreviewResult
package iteration

import (
	"sync"
	"time"

	"a2a-brainstorm/backend/internal/modules/state"
)

// PreviewResult holds the output produced by a single-agent preview dispatch.
// PreviewID is a opaque string token (UUID v4 format) used to verify
// optimistic concurrency on Apply.
type PreviewResult struct {
	// PreviewID is generated at Set time and returned to the client.
	// The client must echo it back in POST .../apply for validation.
	PreviewID string

	// AgentID is the UUID string of the agent that produced this preview.
	AgentID string

	// Output is the CanonicalState returned by the single-agent dispatch.
	// It has NOT been merged with the session's live state yet.
	Output state.CanonicalState

	// CreatedAt is the wall-clock time at which the preview was stored.
	CreatedAt time.Time
}

// PreviewStore is a thread-safe in-memory store keyed by sessionID → agentID.
// A zero value is not valid; use NewPreviewStore.
type PreviewStore struct {
	mu sync.RWMutex
	m  map[string]map[string]PreviewResult // sessionID → agentID → result
}

// NewPreviewStore allocates an empty PreviewStore.
func NewPreviewStore() *PreviewStore {
	return &PreviewStore{
		m: make(map[string]map[string]PreviewResult),
	}
}

// Set stores a PreviewResult for the given session/agent pair.
// If a prior preview for the same pair exists it is overwritten.
// The result is returned for convenience (same value that was stored).
func (ps *PreviewStore) Set(sessionID, agentID string, result PreviewResult) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.m[sessionID] == nil {
		ps.m[sessionID] = make(map[string]PreviewResult)
	}
	ps.m[sessionID][agentID] = result
}

// Get retrieves the PreviewResult for the given session/agent pair.
// Returns (result, true) when found; (zero, false) when not found.
func (ps *PreviewStore) Get(sessionID, agentID string) (PreviewResult, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	agents, ok := ps.m[sessionID]
	if !ok {
		return PreviewResult{}, false
	}
	result, ok := agents[agentID]
	return result, ok
}

// Delete removes the preview for a specific agent within a session.
// It is a no-op if no preview exists (idempotent).
func (ps *PreviewStore) Delete(sessionID, agentID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	agents, ok := ps.m[sessionID]
	if !ok {
		return
	}
	delete(agents, agentID)
	if len(agents) == 0 {
		delete(ps.m, sessionID)
	}
}

// Clear removes all previews for every agent in a session.
// Called after the full iteration engine completes a pass, invalidating
// any stale single-agent previews.
func (ps *PreviewStore) Clear(sessionID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.m, sessionID)
}
