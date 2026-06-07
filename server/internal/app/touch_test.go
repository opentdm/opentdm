package app

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTouchBuffer_CoalescesAndClears(t *testing.T) {
	b := newTouchBuffer()
	id := uuid.New()
	t1 := time.Unix(100, 0)
	t2 := time.Unix(200, 0)
	b.recordToken(id, t1)
	b.recordToken(id, t2) // same id → last write wins, still one entry
	pat := uuid.New()
	b.recordPAT(pat, t1)

	tokens, pats := b.drain()
	if len(tokens) != 1 || !tokens[id].Equal(t2) {
		t.Fatalf("tokens = %v, want one entry for %v at %v", tokens, id, t2)
	}
	if len(pats) != 1 || !pats[pat].Equal(t1) {
		t.Fatalf("pats = %v, want one entry for %v", pats, pat)
	}
	// drain clears the buffer (idempotent flush).
	tokens, pats = b.drain()
	if len(tokens) != 0 || len(pats) != 0 {
		t.Fatalf("expected empty after drain, got tokens=%d pats=%d", len(tokens), len(pats))
	}
}

type fakeTouchSink struct {
	mu         sync.Mutex
	tokenIDs   []uuid.UUID
	patIDs     []uuid.UUID
	tokenCalls int
	patCalls   int
}

func (f *fakeTouchSink) TouchTokensBatch(_ context.Context, ids []uuid.UUID, _ []time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokenIDs = append(f.tokenIDs, ids...)
	f.tokenCalls++
	return nil
}

func (f *fakeTouchSink) TouchPATsBatch(_ context.Context, ids []uuid.UUID, _ []time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.patIDs = append(f.patIDs, ids...)
	f.patCalls++
	return nil
}

func TestService_FlushTouches(t *testing.T) {
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fake := &fakeTouchSink{}
	s := &Service{touch: newTouchBuffer(), touchSink: fake, nowFn: func() time.Time { return fixed }}

	tokA := uuid.New()
	patB := uuid.New()
	s.recordTokenUse(tokA)
	s.recordTokenUse(tokA) // coalesces to one
	s.recordPATUse(patB)

	if err := s.FlushTouches(context.Background()); err != nil {
		t.Fatalf("FlushTouches: %v", err)
	}
	if len(fake.tokenIDs) != 1 || fake.tokenIDs[0] != tokA {
		t.Fatalf("flushed token ids = %v, want [%v]", fake.tokenIDs, tokA)
	}
	if len(fake.patIDs) != 1 || fake.patIDs[0] != patB {
		t.Fatalf("flushed pat ids = %v, want [%v]", fake.patIDs, patB)
	}

	// A second flush on an empty buffer issues no further writes.
	if err := s.FlushTouches(context.Background()); err != nil {
		t.Fatalf("second FlushTouches: %v", err)
	}
	if fake.tokenCalls != 1 || fake.patCalls != 1 {
		t.Fatalf("expected one batch call each, got token=%d pat=%d", fake.tokenCalls, fake.patCalls)
	}
}
