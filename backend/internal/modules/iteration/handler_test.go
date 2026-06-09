package iteration

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"a2a-brainstorm/backend/internal/modules/state"
)

type stubIterationHandlerService struct {
	triggerCalled bool
}

func (s *stubIterationHandlerService) TriggerIteration(_ context.Context, _ string, _ string) (IterationResult, error) {
	s.triggerCalled = true
	return IterationResult{}, nil
}

func (s *stubIterationHandlerService) Preview(_ context.Context, _, _ string) (PreviewResponse, error) {
	return PreviewResponse{}, nil
}

func (s *stubIterationHandlerService) Apply(_ context.Context, _, _, _ string) (state.CanonicalState, error) {
	return state.CanonicalState{}, nil
}

func (s *stubIterationHandlerService) DiscardPreview(_ context.Context, _, _ string) error {
	return nil
}

func (s *stubIterationHandlerService) CheckSessionExists(_ context.Context, _ string) error {
	return nil
}

func newIterationTestMux(svc iterationSvc) *http.ServeMux {
	mux := http.NewServeMux()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	NewHandlerWithService(svc, logger).RegisterRoutes(mux)
	return mux
}

func TestHandleIterate_RejectsUnknownFeedbackFields(t *testing.T) {
	svc := &stubIterationHandlerService{}
	mux := newIterationTestMux(svc)

	body := []byte(`{"user_feedback":"keep scope tight","unexpected":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions/00000000-0000-0000-0000-000000000001/iterate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d", w.Code)
	}
	if svc.triggerCalled {
		t.Fatal("expected TriggerIteration not to be called on invalid body")
	}
}

func TestHandleIterate_RejectsOversizedFeedbackBody(t *testing.T) {
	svc := &stubIterationHandlerService{}
	mux := newIterationTestMux(svc)

	largeFeedback := strings.Repeat("a", (1<<20)+256)
	body := []byte(`{"user_feedback":"` + largeFeedback + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions/00000000-0000-0000-0000-000000000001/iterate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized body, got %d", w.Code)
	}
	if svc.triggerCalled {
		t.Fatal("expected TriggerIteration not to be called on oversized body")
	}
}
