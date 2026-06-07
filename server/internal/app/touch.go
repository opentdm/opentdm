package app

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/store"
)

// maxTouchBuffer is a soft cap on the pending set. Reaching it signals an early
// flush so a burst of distinct credentials can't grow the maps unbounded
// between ticks. Dedup by id already keeps the size near the active-credential
// count in practice.
const maxTouchBuffer = 10_000

// touchSink persists coalesced last-used timestamps. It is defined here, at the
// point of use, so the flusher is unit-testable with a fake (no DB).
type touchSink interface {
	TouchTokensBatch(ctx context.Context, ids []uuid.UUID, ats []time.Time) error
	TouchPATsBatch(ctx context.Context, ids []uuid.UUID, ats []time.Time) error
}

// storeTouchSink adapts *store.Store to touchSink.
type storeTouchSink struct{ st *store.Store }

func (s storeTouchSink) TouchTokensBatch(ctx context.Context, ids []uuid.UUID, ats []time.Time) error {
	return s.st.Q().TouchTokensBatch(ctx, ids, ats)
}

func (s storeTouchSink) TouchPATsBatch(ctx context.Context, ids []uuid.UUID, ats []time.Time) error {
	return s.st.Q().TouchUserPATsBatch(ctx, ids, ats)
}

// touchBuffer coalesces service-token and PAT last-used timestamps in memory.
// The request hot path only takes a lock and sets a map entry — never a DB
// write — fulfilling the "no per-request last_used writes" decision. Last write
// wins within a flush interval.
type touchBuffer struct {
	mu     sync.Mutex
	tokens map[uuid.UUID]time.Time
	pats   map[uuid.UUID]time.Time
	wake   chan struct{} // buffered(1): a full buffer nudges an early flush
}

func newTouchBuffer() *touchBuffer {
	return &touchBuffer{
		tokens: make(map[uuid.UUID]time.Time),
		pats:   make(map[uuid.UUID]time.Time),
		wake:   make(chan struct{}, 1),
	}
}

func (b *touchBuffer) recordToken(id uuid.UUID, at time.Time) {
	b.mu.Lock()
	b.tokens[id] = at
	full := len(b.tokens)+len(b.pats) >= maxTouchBuffer
	b.mu.Unlock()
	if full {
		b.signal()
	}
}

func (b *touchBuffer) recordPAT(id uuid.UUID, at time.Time) {
	b.mu.Lock()
	b.pats[id] = at
	full := len(b.tokens)+len(b.pats) >= maxTouchBuffer
	b.mu.Unlock()
	if full {
		b.signal()
	}
}

func (b *touchBuffer) signal() {
	select {
	case b.wake <- struct{}{}:
	default: // a wake is already pending
	}
}

// drain atomically returns and clears the pending sets, making flush idempotent.
func (b *touchBuffer) drain() (tokens, pats map[uuid.UUID]time.Time) {
	b.mu.Lock()
	tokens, pats = b.tokens, b.pats
	b.tokens = make(map[uuid.UUID]time.Time)
	b.pats = make(map[uuid.UUID]time.Time)
	b.mu.Unlock()
	return tokens, pats
}

// recordTokenUse / recordPATUse buffer a best-effort last-used timestamp. They
// never touch the DB and never error the request.
func (s *Service) recordTokenUse(id uuid.UUID) { s.touch.recordToken(id, s.nowFn()) }
func (s *Service) recordPATUse(id uuid.UUID)   { s.touch.recordPAT(id, s.nowFn()) }

// FlushTouches writes all buffered last-used timestamps. It is idempotent: drain
// clears the buffer, so a redundant call (e.g. ticker + shutdown) is a no-op.
func (s *Service) FlushTouches(ctx context.Context) error {
	tokens, pats := s.touch.drain()
	var firstErr error
	if len(tokens) > 0 {
		ids, ats := splitTouch(tokens)
		if err := s.touchSink.TouchTokensBatch(ctx, ids, ats); err != nil {
			firstErr = err
		}
	}
	if len(pats) > 0 {
		ids, ats := splitTouch(pats)
		if err := s.touchSink.TouchPATsBatch(ctx, ids, ats); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// StartTouchFlusher flushes buffered timestamps on an interval until ctx is
// cancelled, then performs one final flush with a fresh context so a graceful
// shutdown doesn't drop the last interval of activity. Errors are best-effort
// (last_used_at is advisory), so they are not surfaced.
func (s *Service) StartTouchFlusher(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				_ = s.FlushTouches(context.Background())
				return
			case <-t.C:
				_ = s.FlushTouches(ctx)
			case <-s.touch.wake:
				_ = s.FlushTouches(ctx)
			}
		}
	}()
}

func splitTouch(m map[uuid.UUID]time.Time) (ids []uuid.UUID, ats []time.Time) {
	ids = make([]uuid.UUID, 0, len(m))
	ats = make([]time.Time, 0, len(m))
	for id, at := range m {
		ids = append(ids, id)
		ats = append(ats, at)
	}
	return ids, ats
}
