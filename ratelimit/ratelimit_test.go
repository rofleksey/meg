package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterSeparatesKeysAndRefillsTokens(t *testing.T) {
	limiter := NewLimiter(2, time.Minute)
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)

	limiter.now = func() time.Time { return now }
	for range 2 {
		if ok, _ := limiter.Allow("client-a"); !ok {
			t.Fatal("request unexpectedly rejected")
		}
	}

	if ok, retry := limiter.Allow("client-a"); ok || retry <= 0 {
		t.Fatalf("limit = (%v, %s), want rejection with retry", ok, retry)
	}

	if ok, _ := limiter.Allow("client-b"); !ok {
		t.Fatal("second client shares a budget")
	}

	now = now.Add(30 * time.Second)

	if ok, _ := limiter.Allow("client-a"); !ok {
		t.Fatal("token was not refilled")
	}
}

func TestLimiterBoundsTrackedKeys(t *testing.T) {
	limiter := NewLimiter(1, time.Minute)
	limiter.now = func() time.Time { return time.Unix(0, 0) }

	for index := range maxTrackedKeys + 1 {
		limiter.Allow(string(rune(index + 1)))
	}

	if got := len(limiter.entries); got != maxTrackedKeys {
		t.Fatalf("tracked keys = %d, want %d", got, maxTrackedKeys)
	}
}
