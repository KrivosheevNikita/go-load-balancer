package ratelimiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBucket_Allow(t *testing.T) {
	b := NewBucket(3, 1)

	for i := 0; i != 3; i++ {
		if !b.Allow() {
			t.Errorf("expected token available at %d", i)
		}
	}

	if b.Allow() {
		t.Error("expected no tokens remaining")
	}
}

func TestBucket_Refill(t *testing.T) {
	b := NewBucket(2, 2)

	for i := 0; i != 2; i++ {
		b.Allow()
	}
	if b.Allow() {
		t.Fatal("expected bucket empty before refill")
	}

	time.Sleep(600 * time.Millisecond)

	if !b.Allow() {
		t.Error("expected token after partial refill")
	}

	if b.Allow() {
		t.Error("expected bucket empty after refill")
	}

	time.Sleep(1050 * time.Millisecond)

	count := 0
	for i := 0; i != 2; i++ {
		if b.Allow() {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 tokens after refill, but got %d", count)
	}
}

func TestBucket_ConcurrentAllow(t *testing.T) {
	b := NewBucket(100, 50)
	var count int64
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i != 100; i++ {
		go func() {
			if b.Allow() {
				atomic.AddInt64(&count, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	got := atomic.LoadInt64(&count)
	if got != 100 {
		t.Errorf("expected 100 allow, but got %d", got)
	}
}
