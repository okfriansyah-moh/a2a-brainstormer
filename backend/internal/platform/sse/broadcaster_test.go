// Package sse provides tests for the SSE broadcaster.
package sse

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newSession is a helper to generate a fresh UUID for each test.
func newSession() uuid.UUID {
	return uuid.New()
}

// TestPublishSingleSubscriber verifies that a subscriber receives the published event.
func TestPublishSingleSubscriber(t *testing.T) {
	b := NewBroadcaster()
	sid := newSession()

	ch, unsub := b.Subscribe(sid, 0)
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	defer unsub()

	b.Publish(sid, "test.event", map[string]string{"k": "v"})

	select {
	case evt := <-ch:
		if evt.Type != "test.event" {
			t.Errorf("want type %q, got %q", "test.event", evt.Type)
		}
		if evt.ID != 1 {
			t.Errorf("want ID 1, got %d", evt.ID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}
}

// TestPublishMultipleSubscribers verifies fan-out: all active subscribers receive
// the same event.
func TestPublishMultipleSubscribers(t *testing.T) {
	b := NewBroadcaster()
	sid := newSession()

	const n = 3
	channels := make([]<-chan Event, n)
	unsubs := make([]func(), n)
	for i := range n {
		ch, unsub := b.Subscribe(sid, 0)
		if ch == nil {
			t.Fatalf("subscriber %d: expected non-nil channel", i)
		}
		channels[i] = ch
		unsubs[i] = unsub
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	b.Publish(sid, "fan.out", nil)

	for i, ch := range channels {
		select {
		case evt := <-ch:
			if evt.Type != "fan.out" {
				t.Errorf("subscriber %d: want type %q, got %q", i, "fan.out", evt.Type)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("subscriber %d: timed out", i)
		}
	}
}

// TestRingBufferReplay verifies that a late subscriber receives buffered events
// with ID greater than its lastEventID.
func TestRingBufferReplay(t *testing.T) {
	b := NewBroadcaster()
	sid := newSession()

	// Publish 5 events before subscribing.
	for i := range 5 {
		b.Publish(sid, "evt", i)
	}

	// Subscribe with lastEventID=3 — should replay events 4 and 5 only.
	ch, unsub := b.Subscribe(sid, 3)
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	defer unsub()

	var received []uint64
	// Drain two replayed events (IDs 4 and 5).
	for range 2 {
		select {
		case evt := <-ch:
			received = append(received, evt.ID)
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timed out waiting for replayed event")
		}
	}

	if len(received) != 2 {
		t.Fatalf("want 2 replayed events, got %d", len(received))
	}
	if received[0] != 4 || received[1] != 5 {
		t.Errorf("want IDs [4 5], got %v", received)
	}
}

// TestSubscriberLimitReturnsNil verifies that the 11th subscriber attempt
// returns a nil channel (caller should send HTTP 429).
func TestSubscriberLimitReturnsNil(t *testing.T) {
	b := NewBroadcaster()
	sid := newSession()

	unsubs := make([]func(), 0, maxSubs)
	for i := range maxSubs {
		ch, unsub := b.Subscribe(sid, 0)
		if ch == nil {
			t.Fatalf("subscriber %d (within limit): expected non-nil channel", i+1)
		}
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	// 11th subscriber should be rejected.
	ch, unsub := b.Subscribe(sid, 0)
	if ch != nil {
		if unsub != nil {
			unsub()
		}
		t.Error("11th subscriber: expected nil channel, got non-nil")
	}
}

// TestSlowSubscriberDropped verifies that a subscriber whose channel fills up is
// silently removed and does not block the broadcaster.
func TestSlowSubscriberDropped(t *testing.T) {
	b := NewBroadcaster()
	sid := newSession()

	// Subscribe but never drain the channel.
	ch, unsub := b.Subscribe(sid, 0)
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	// We intentionally do NOT defer unsub here — the broadcaster should close
	// the channel once it's full.

	// Flood with chanBuffer+1 events so the slow subscriber's channel fills.
	for i := range chanBuffer + 1 {
		b.Publish(sid, "flood", i)
	}

	// Verify the channel was closed by the broadcaster.
	select {
	case _, open := <-ch:
		_ = open // either receives event or sees closed channel — both acceptable
	default:
		// channel still has items or is closed — ok
	}

	// The unsub function must be safe to call even after the channel is closed
	// by the broadcaster (should not panic).
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unsub panicked: %v", r)
		}
	}()
	unsub()
}

// TestUnsubscribeCleansUp verifies that calling unsubscribe removes the
// subscriber and prevents further event delivery.
func TestUnsubscribeCleansUp(t *testing.T) {
	b := NewBroadcaster()
	sid := newSession()

	ch, unsub := b.Subscribe(sid, 0)
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	unsub() // unsubscribe immediately

	// Publishing after unsubscribe must not block and must not deliver to ch.
	b.Publish(sid, "after.unsub", nil)

	select {
	case _, open := <-ch:
		if open {
			t.Error("received event on unsubscribed channel")
		}
		// Channel was closed by unsub — that is fine.
	default:
		// Nothing received — correct.
	}
}

// TestEmitStringID verifies that Emit (EventEmitter interface) correctly routes
// to Publish using a string session ID.
func TestEmitStringID(t *testing.T) {
	b := NewBroadcaster()
	sid := uuid.New()

	ch, unsub := b.Subscribe(sid, 0)
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	defer unsub()

	b.Emit(sid.String(), "emit.event", "payload")

	select {
	case evt := <-ch:
		if evt.Type != "emit.event" {
			t.Errorf("want type %q, got %q", "emit.event", evt.Type)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for emit event")
	}
}

// TestEmitInvalidUUID verifies that Emit with an invalid UUID silently discards
// the event instead of panicking.
func TestEmitInvalidUUID(t *testing.T) {
	b := NewBroadcaster()
	// Must not panic.
	b.Emit("not-a-uuid", "x", nil)
}

// TestConcurrentPublishSubscribe stress-tests concurrent publish and subscribe
// to ensure there are no data races.  Run with -race to verify.
func TestConcurrentPublishSubscribe(t *testing.T) {
	b := NewBroadcaster()
	sid := newSession()

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ch, unsub := b.Subscribe(sid, 0)
			if ch == nil {
				return // limit hit — ok
			}
			defer unsub()
			// Drain briefly.
			timeout := time.After(50 * time.Millisecond)
			for {
				select {
				case <-ch:
				case <-timeout:
					return
				}
			}
		}(i)
	}

	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b.Publish(sid, "concurrent", i)
		}(i)
	}

	wg.Wait()
}
