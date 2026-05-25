// Package iteration — preview_test.go tests the PreviewStore and the
// service-layer Preview/Apply/DiscardPreview methods.
//
// All tests use in-memory stubs — no live DB, no live A2A endpoint.
package iteration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"a2a-brainstorm/backend/internal/modules/session"
	"a2a-brainstorm/backend/internal/modules/state"
)

// ─── PreviewStore unit tests ──────────────────────────────────────────────────

func TestPreviewStore_SetAndGet(t *testing.T) {
	ps := NewPreviewStore()
	sid, aid := "session-1", "agent-1"

	result := PreviewResult{
		PreviewID: "preview-1",
		AgentID:   aid,
		Output:    state.CanonicalState{},
		CreatedAt: time.Now().UTC(),
	}
	ps.Set(sid, aid, result)

	got, ok := ps.Get(sid, aid)
	if !ok {
		t.Fatal("expected preview to be found after Set")
	}
	if got.PreviewID != "preview-1" {
		t.Errorf("got preview_id %q, want %q", got.PreviewID, "preview-1")
	}
}

func TestPreviewStore_GetMissing(t *testing.T) {
	ps := NewPreviewStore()
	_, ok := ps.Get("no-session", "no-agent")
	if ok {
		t.Error("expected Get to return false for unknown key")
	}
}

func TestPreviewStore_Delete(t *testing.T) {
	ps := NewPreviewStore()
	ps.Set("s1", "a1", PreviewResult{PreviewID: "p1"})
	ps.Delete("s1", "a1")

	_, ok := ps.Get("s1", "a1")
	if ok {
		t.Error("expected preview to be absent after Delete")
	}
}

func TestPreviewStore_Delete_Idempotent(t *testing.T) {
	ps := NewPreviewStore()
	// Delete of a non-existent entry must not panic.
	ps.Delete("ghost-session", "ghost-agent")
}

func TestPreviewStore_Clear(t *testing.T) {
	ps := NewPreviewStore()
	ps.Set("s1", "a1", PreviewResult{PreviewID: "p1"})
	ps.Set("s1", "a2", PreviewResult{PreviewID: "p2"})
	ps.Set("s2", "a1", PreviewResult{PreviewID: "p3"})

	ps.Clear("s1")

	if _, ok := ps.Get("s1", "a1"); ok {
		t.Error("expected s1/a1 to be cleared")
	}
	if _, ok := ps.Get("s1", "a2"); ok {
		t.Error("expected s1/a2 to be cleared")
	}
	// s2 must be unaffected.
	if _, ok := ps.Get("s2", "a1"); !ok {
		t.Error("s2/a1 should not be affected by Clear(s1)")
	}
}

func TestPreviewStore_Concurrent(t *testing.T) {
	ps := NewPreviewStore()
	const n = 50

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sid := "session"
			aid := "agent"
			ps.Set(sid, aid, PreviewResult{PreviewID: "p"})
			ps.Get(sid, aid)
			if i%5 == 0 {
				ps.Delete(sid, aid)
			}
		}(i)
	}
	wg.Wait()
}

// ─── sessionLockMap unit tests ────────────────────────────────────────────────

func TestSessionLockMap_GetOrCreate(t *testing.T) {
	m := newSessionLockMap()
	l1 := m.getLock("sess-1")
	l2 := m.getLock("sess-1")
	if l1 != l2 {
		t.Error("same sessionID must return the same mutex pointer")
	}
	l3 := m.getLock("sess-2")
	if l1 == l3 {
		t.Error("different sessionIDs must return different mutex pointers")
	}
}

func TestSessionLockMap_TryLock(t *testing.T) {
	m := newSessionLockMap()
	lock := m.getLock("sess-try")
	if !lock.TryLock() {
		t.Fatal("initial TryLock should succeed")
	}
	// A second TryLock while held must fail.
	if lock.TryLock() {
		lock.Unlock() // clean up
		t.Fatal("second TryLock on held mutex should return false")
	}
	lock.Unlock()
}

// ─── Service integration stubs ────────────────────────────────────────────────

// stubSessionProvider returns a hard-coded session or an error.
type stubSessionProvider struct {
	sess session.Session
	err  error
}

func (s *stubSessionProvider) GetSession(_ context.Context, _ string) (session.Session, error) {
	return s.sess, s.err
}

// stubStateWriter records the last UpdateState call.
type stubStateWriter struct {
	called bool
	last   state.CanonicalState
	err    error
}

func (s *stubStateWriter) UpdateState(_ context.Context, _ string, cs *state.CanonicalState) error {
	s.called = true
	if cs != nil {
		s.last = *cs
	}
	return s.err
}

func (s *stubStateWriter) UpdateStatus(_ context.Context, _ string, _ string) error {
	return nil
}

// ─── Service.DiscardPreview ───────────────────────────────────────────────────

func TestService_DiscardPreview_Idempotent(t *testing.T) {
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		logger:       testLogger(),
	}
	// No preview exists — discard must succeed silently.
	if err := svc.DiscardPreview(context.Background(), "s1", "a1"); err != nil {
		t.Fatalf("DiscardPreview returned unexpected error: %v", err)
	}
}

func TestService_DiscardPreview_RemovesExisting(t *testing.T) {
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		logger:       testLogger(),
	}
	svc.previews.Set("s1", "a1", PreviewResult{PreviewID: "p1"})

	if err := svc.DiscardPreview(context.Background(), "s1", "a1"); err != nil {
		t.Fatalf("DiscardPreview returned unexpected error: %v", err)
	}
	if _, ok := svc.previews.Get("s1", "a1"); ok {
		t.Error("preview should be gone after DiscardPreview")
	}
}

// ─── Service.Apply — error paths ─────────────────────────────────────────────

func TestService_Apply_NoPreview_ReturnsErrPreviewNotFound(t *testing.T) {
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		sessions:     &stubSessionProvider{sess: session.Session{ID: "s1"}},
		store:        &stubStateWriter{},
		logger:       testLogger(),
	}
	_, err := svc.Apply(context.Background(), "s1", "a1", "")
	if !errors.Is(err, ErrPreviewNotFound) {
		t.Errorf("expected ErrPreviewNotFound, got %v", err)
	}
}

func TestService_Apply_WrongPreviewID_ReturnsErrPreviewIDMismatch(t *testing.T) {
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		sessions:     &stubSessionProvider{sess: session.Session{ID: "s1"}},
		store:        &stubStateWriter{},
		logger:       testLogger(),
	}
	svc.previews.Set("s1", "a1", PreviewResult{PreviewID: "correct-id"})

	_, err := svc.Apply(context.Background(), "s1", "a1", "wrong-id")
	if !errors.Is(err, ErrPreviewIDMismatch) {
		t.Errorf("expected ErrPreviewIDMismatch, got %v", err)
	}
}

func TestService_Apply_EmptyPreviewID_SkipsGuard(t *testing.T) {
	writer := &stubStateWriter{}
	sess := session.Session{
		ID:   "s1",
		Idea: "test",
		CurrentState: &state.CanonicalState{
			Meta: state.StateMeta{Iteration: 2},
		},
	}
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		sessions:     &stubSessionProvider{sess: sess},
		store:        writer,
		logger:       testLogger(),
	}
	svc.previews.Set("s1", "a1", PreviewResult{
		PreviewID: "some-id",
		Output:    state.CanonicalState{},
	})

	got, err := svc.Apply(context.Background(), "s1", "a1", "")
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if got.Meta.Iteration != 3 {
		t.Errorf("expected iteration 3, got %d", got.Meta.Iteration)
	}
	if !writer.called {
		t.Error("expected UpdateState to be called")
	}
	// Preview must be cleared after apply.
	if _, ok := svc.previews.Get("s1", "a1"); ok {
		t.Error("preview should be cleared after Apply")
	}
}

func TestService_Apply_CorrectPreviewID_Applies(t *testing.T) {
	writer := &stubStateWriter{}
	sess := session.Session{
		ID:           "s1",
		Idea:         "test",
		CurrentState: &state.CanonicalState{Meta: state.StateMeta{Iteration: 1}},
	}
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		sessions:     &stubSessionProvider{sess: sess},
		store:        writer,
		logger:       testLogger(),
	}
	svc.previews.Set("s1", "a1", PreviewResult{
		PreviewID: "good-id",
		Output:    state.CanonicalState{},
	})

	_, err := svc.Apply(context.Background(), "s1", "a1", "good-id")
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if !writer.called {
		t.Error("expected UpdateState to be called")
	}
}

// ─── ErrIterationInFlight ────────────────────────────────────────────────────

func TestService_TriggerIteration_ReturnsBusy_WhenLocked(t *testing.T) {
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		sessions:     &stubSessionProvider{sess: session.Session{ID: "s1"}},
		store:        &stubStateWriter{},
		logger:       testLogger(),
	}
	// Acquire the lock directly to simulate an in-flight operation.
	lock := svc.sessionLocks.getLock("s1")
	lock.Lock()
	defer lock.Unlock()

	_, err := svc.TriggerIteration(context.Background(), "s1")
	if !errors.Is(err, ErrIterationInFlight) {
		t.Errorf("expected ErrIterationInFlight, got %v", err)
	}
}

func TestService_Preview_ReturnsBusy_WhenLocked(t *testing.T) {
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		sessions:     &stubSessionProvider{sess: session.Session{ID: "s1"}},
		store:        &stubStateWriter{},
		logger:       testLogger(),
	}
	lock := svc.sessionLocks.getLock("s1")
	lock.Lock()
	defer lock.Unlock()

	_, err := svc.Preview(context.Background(), "s1", "a1")
	if !errors.Is(err, ErrIterationInFlight) {
		t.Errorf("expected ErrIterationInFlight, got %v", err)
	}
}

func TestService_Apply_ReturnsBusy_WhenLocked(t *testing.T) {
	svc := &Service{
		previews:     NewPreviewStore(),
		sessionLocks: newSessionLockMap(),
		sessions:     &stubSessionProvider{sess: session.Session{ID: "s1"}},
		store:        &stubStateWriter{},
		logger:       testLogger(),
	}
	lock := svc.sessionLocks.getLock("s1")
	lock.Lock()
	defer lock.Unlock()

	_, err := svc.Apply(context.Background(), "s1", "a1", "")
	if !errors.Is(err, ErrIterationInFlight) {
		t.Errorf("expected ErrIterationInFlight, got %v", err)
	}
}
