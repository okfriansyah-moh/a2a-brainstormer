package ctxidle

import (
	"context"
	"testing"
	"time"
)

func TestWithIdleTimeout_NoCancelBeforeFirstBump(t *testing.T) {
	ctx, _ := WithIdleTimeout(context.Background(), 50*time.Millisecond)
	time.Sleep(120 * time.Millisecond)
	if ctx.Err() != nil {
		t.Fatalf("expected no cancel before first bump, got %v", ctx.Err())
	}
}

func TestWithIdleTimeout_CancelsOnIdleAfterBump(t *testing.T) {
	ctx, bump := WithIdleTimeout(context.Background(), 50*time.Millisecond)
	bump()
	<-ctx.Done()
	if ctx.Err() != context.Canceled {
		t.Fatalf("expected canceled, got %v", ctx.Err())
	}
}

func TestWithIdleTimeout_BumpResetsIdle(t *testing.T) {
	ctx, bump := WithIdleTimeout(context.Background(), 80*time.Millisecond)
	bump()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		bump()
		time.Sleep(30 * time.Millisecond)
		if ctx.Err() != nil {
			t.Fatal("context cancelled while bumping idle timer")
		}
	}
}

func TestWithIdleTimeout_ParentCancel(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	ctx, bump := WithIdleTimeout(parent, time.Minute)
	bump()
	parentCancel()
	<-ctx.Done()
}
