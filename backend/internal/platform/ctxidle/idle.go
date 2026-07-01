// Package ctxidle provides a context that cancels after a period of inactivity.
// The idle timer arms on the first Bump() call so callers can wait indefinitely
// for time-to-first-token while still aborting stalled streams mid-output.
package ctxidle

import (
	"context"
	"sync"
	"time"
)

type idleCtx struct {
	context.Context
	idle         time.Duration
	startup      time.Duration
	timer        *time.Timer
	startupTimer *time.Timer
	cancel       context.CancelFunc
	mu           sync.Mutex
	armed        bool
}

// WithIdleTimeout returns a child context that is cancelled when idle elapses
// without a Bump after the timer is armed. The timer is NOT started until the
// first Bump — so a long LLM think phase before the first token does not trip
// the idle deadline. Subsequent Bump calls reset the timer.
func WithIdleTimeout(parent context.Context, idle time.Duration) (context.Context, func()) {
	return WithStartupAndIdleTimeout(parent, 0, idle)
}

// WithStartupAndIdleTimeout cancels when Bump is not called within startup
// (time-to-first-token), then enforces idle between subsequent Bump calls.
// Pass startup=0 to disable the first-token deadline.
func WithStartupAndIdleTimeout(parent context.Context, startup, idle time.Duration) (context.Context, func()) {
	if idle <= 0 && startup <= 0 {
		return parent, func() {}
	}

	ctx, cancel := context.WithCancel(parent)
	ic := &idleCtx{
		Context: ctx,
		idle:    idle,
		startup: startup,
		cancel:  cancel,
	}

	if startup > 0 {
		ic.startupTimer = time.AfterFunc(startup, func() {
			ic.cancel()
		})
	}

	bump := func() {
		ic.mu.Lock()
		defer ic.mu.Unlock()
		if ic.startupTimer != nil {
			ic.startupTimer.Stop()
			ic.startupTimer = nil
		}
		if ic.idle <= 0 {
			return
		}
		if ic.armed {
			if ic.timer != nil {
				ic.timer.Reset(ic.idle)
			}
			return
		}
		ic.armed = true
		ic.timer = time.AfterFunc(ic.idle, func() {
			ic.cancel()
		})
	}

	return ic, bump
}

func (ic *idleCtx) Done() <-chan struct{} {
	return ic.Context.Done()
}

func (ic *idleCtx) Err() error {
	return ic.Context.Err()
}

func (ic *idleCtx) Value(key any) any {
	return ic.Context.Value(key)
}
