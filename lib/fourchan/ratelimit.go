package fourchan

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RequestInterval is the minimum spacing between any two requests made
// through a Limiter — 4chan's API requires 1 req/s, shared globally across
// every board and every endpoint, not per-board.
const RequestInterval = time.Second

// jitterTolerance absorbs ordinary scheduling slack between rl.Wait
// returning and time.Now() being read, so routine ~1ms jitter doesn't
// trip the violation warning below.
const jitterTolerance = 50 * time.Millisecond

// Limiter enforces RequestInterval across every call that shares it. There
// must be exactly one Limiter per process; passing separate Limiters to
// separate boards would silently recreate a per-board bucket and violate
// the 1 req/s ceiling.
type Limiter struct {
	rl *rate.Limiter

	mu   sync.Mutex
	last time.Time
}

// NewLimiter returns a Limiter enforcing RequestInterval spacing.
func NewLimiter() *Limiter {
	return &Limiter{rl: rate.NewLimiter(rate.Every(RequestInterval), 1)}
}

// Wait blocks until the caller is allowed to issue its next request. It
// logs loudly if two requests somehow went out closer together than
// RequestInterval — that should be structurally impossible through this
// Limiter, so seeing it means something bypassed it.
func (l *Limiter) Wait(ctx context.Context) error {
	if err := l.rl.Wait(ctx); err != nil {
		return err
	}

	l.mu.Lock()
	now := time.Now()
	if !l.last.IsZero() {
		if gap := now.Sub(l.last); gap < RequestInterval-jitterTolerance {
			slog.Warn("rate limiter ceiling violated", "gap", gap, "want_min", RequestInterval)
		}
	}
	l.last = now
	l.mu.Unlock()

	return nil
}
