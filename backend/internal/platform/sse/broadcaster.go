// Package sse provides the server-sent events broadcaster used by the
// iteration pipeline to stream real-time agent lifecycle events to connected
// frontend clients.
//
// Design constraints from §8.22 of docs/PLAN.md:
//   - Ring buffer cap:          100 events per session.
//   - Max subscribers/session:  10 (11th Subscribe returns nil → caller sends 429).
//   - Subscriber channel buf:   32. When full the subscriber is dropped silently
//     (treated as disconnected — fire-and-forget semantics).
//   - Last-Event-ID replay:     On Subscribe, events with ID > lastEventID that
//     are still in the ring buffer are replayed before live events begin.
//
// EventEmitter interface is defined here so both the iteration engine and the
// session service can fire events without importing the full Broadcaster type.
// *Broadcaster and NoopEmitter both satisfy EventEmitter.
package sse

import (
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

const (
	ringCap    = 100 // ring buffer capacity per session
	maxSubs    = 10  // max subscribers per session
	chanBuffer = 32  // per-subscriber channel buffer
)

// Event is the unit of communication between the broadcaster and SSE handlers.
// ID is a monotonically increasing per-session counter (starts at 1).
// Type maps to the SSE `event:` field. Data is serialised as JSON in `data:`.
type Event struct {
	ID   uint64
	Type string
	Data any
}

// EventEmitter is the narrow interface used by the iteration engine and session
// service to fire events without importing the full Broadcaster.
// Satisfied by *Broadcaster and NoopEmitter.
//
// Emit takes a string session ID so callers (iteration engine, session service)
// do not need to import uuid just to fire events.
type EventEmitter interface {
	Emit(sessionID, evtType string, data any)
}

// NoopEmitter is a no-op EventEmitter used in tests and in environments where
// SSE is disabled. All Emit calls are discarded immediately.
type NoopEmitter struct{}

// Emit implements EventEmitter — discards the event.
func (NoopEmitter) Emit(_, _ string, _ any) {}

// Broadcaster distributes events to all active SSE subscribers for each
// session and maintains a per-session ring buffer for Last-Event-ID replay.
type Broadcaster struct {
	mu      sync.RWMutex
	subs    map[uuid.UUID]map[uint64]chan Event // sessionID → subID → channel
	buffers map[uuid.UUID][]Event               // sessionID → ring buffer (cap 100)
	nextEvt map[uuid.UUID]*uint64               // sessionID → next event ID counter
	nextSub uint64                              // global subscriber ID counter (guarded by mu)
}

// NewBroadcaster allocates an empty Broadcaster ready for use.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subs:    make(map[uuid.UUID]map[uint64]chan Event),
		buffers: make(map[uuid.UUID][]Event),
		nextEvt: make(map[uuid.UUID]*uint64),
	}
}

// Publish creates an event, appends it to the session ring buffer, and fans it
// out to all active subscribers for that session.
//
// If a subscriber's channel is full, that subscriber is removed (treated as
// disconnected). Publish never blocks.
func (b *Broadcaster) Publish(sessionID uuid.UUID, evtType string, data any) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ctr := b.getOrInitCounter(sessionID)
	id := atomic.AddUint64(ctr, 1)

	evt := Event{ID: id, Type: evtType, Data: data}

	// Append to ring buffer; trim to cap.
	buf := b.buffers[sessionID]
	buf = append(buf, evt)
	if len(buf) > ringCap {
		buf = buf[len(buf)-ringCap:]
	}
	b.buffers[sessionID] = buf

	// Fan out to all subscribers — non-blocking.
	for subID, ch := range b.subs[sessionID] {
		select {
		case ch <- evt:
		default:
			// Channel full → subscriber is too slow; drop it.
			close(ch)
			delete(b.subs[sessionID], subID)
		}
	}
}

// Emit implements EventEmitter. It accepts a string session ID for ergonomics —
// callers (engine, session service) work with string UUIDs and should not need
// to import uuid.  If sessionID is not a valid UUID the call is silently
// discarded.
func (b *Broadcaster) Emit(sessionID, evtType string, data any) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return
	}
	b.Publish(id, evtType, data)
}

// Subscribe registers a new subscriber for the given session and immediately
// replays any buffered events with ID > lastEventID.
//
// Returns a receive-only channel that receives future events and an unsubscribe
// function that must be called when the client disconnects. The caller MUST
// call unsubscribe to prevent resource leaks.
//
// Returns (nil, nil) when the session already has maxSubs (10) active
// subscribers — the caller should respond with HTTP 429.
func (b *Broadcaster) Subscribe(sessionID uuid.UUID, lastEventID uint64) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.subs[sessionID]) >= maxSubs {
		return nil, nil
	}

	b.nextSub++
	subID := b.nextSub

	ch := make(chan Event, chanBuffer)

	if b.subs[sessionID] == nil {
		b.subs[sessionID] = make(map[uint64]chan Event)
	}
	b.subs[sessionID][subID] = ch

	// Replay missed events from the ring buffer while still holding the lock so
	// we don't race with a concurrent Publish between replay and live stream.
	for _, evt := range b.buffers[sessionID] {
		if evt.ID > lastEventID {
			select {
			case ch <- evt:
			default:
				// Fresh channel; should not fill, but guard just in case.
			}
		}
	}

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs, ok := b.subs[sessionID]
		if !ok {
			return
		}
		if _, exists := subs[subID]; exists {
			close(ch)
			delete(subs, subID)
		}
		if len(subs) == 0 {
			delete(b.subs, sessionID)
		}
	}

	return ch, unsubscribe
}

// getOrInitCounter returns the event-ID counter for sessionID, creating one at
// zero if it does not yet exist.  Caller must hold b.mu.Lock().
func (b *Broadcaster) getOrInitCounter(sessionID uuid.UUID) *uint64 {
	if ctr, ok := b.nextEvt[sessionID]; ok {
		return ctr
	}
	var zero uint64
	b.nextEvt[sessionID] = &zero
	return &zero
}
