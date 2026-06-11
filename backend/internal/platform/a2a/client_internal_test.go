package a2a

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
)

func TestIsTransientError_DeadlineExceeded(t *testing.T) {
	if !isTransientError(context.DeadlineExceeded) {
		t.Fatal("expected context deadline exceeded to be treated as transient")
	}
}

type flakyCardResolver struct {
	calls int
}

func (f *flakyCardResolver) Resolve(ctx context.Context, baseURL string, _ ...agentcard.ResolveOption) (*a2a.AgentCard, error) {
	f.calls++
	if f.calls == 1 {
		return nil, context.DeadlineExceeded
	}
	return agentcard.DefaultResolver.Resolve(ctx, baseURL)
}

func TestNewClient_RetriesTransientResolverError(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent-card.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"retry-agent","description":"retry","version":"1.0","url":"` + srv.URL + `","supportedInterfaces":[{"url":"` + srv.URL + `/","protocolVersion":"1.0","protocolBinding":"HTTP+JSON"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	oldResolver := cardResolver
	defer func() { cardResolver = oldResolver }()

	flaky := &flakyCardResolver{}
	cardResolver = flaky

	client, err := NewClient(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("NewClient returned error after retry: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient returned nil client")
	}
	if flaky.calls < 2 {
		t.Fatalf("expected at least 2 resolver attempts, got %d", flaky.calls)
	}
}
