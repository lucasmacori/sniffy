package ratelimiter

import (
	"context"
	"testing"
	"time"
)

func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(AuthenticatedRate)

	if tb.rate != AuthenticatedRate {
		t.Errorf("rate = %f; want %f", tb.rate, AuthenticatedRate)
	}
	if tb.capacity != TokenCapacity {
		t.Errorf("capacity = %f; want %f", tb.capacity, TokenCapacity)
	}
	if tb.tokens != TokenCapacity {
		t.Errorf("tokens = %f; want %f", tb.tokens, TokenCapacity)
	}
}

func TestTokenBucket_Wait(t *testing.T) {
	t.Run("immediate availability", func(t *testing.T) {
		tb := NewTokenBucket(1000) // very high rate
		ctx := context.Background()

		start := time.Now()
		err := tb.Wait(ctx)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
		if elapsed > 10*time.Millisecond {
			t.Errorf("Wait() took too long: %v", elapsed)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		tb := NewTokenBucket(0.001) // very low rate
		tb.tokens = 0               // exhaust tokens

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := tb.Wait(ctx)
		if err != context.DeadlineExceeded {
			t.Errorf("Wait() error = %v; want context.DeadlineExceeded", err)
		}
	})

	t.Run("token refill", func(t *testing.T) {
		tb := NewTokenBucket(10) // 10 tokens per second
		tb.tokens = 0
		tb.lastRefill = time.Now().Add(-time.Second)

		ctx := context.Background()
		err := tb.Wait(ctx)
		if err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
	})
}

func TestTokenBucket_SetRate(t *testing.T) {
	tb := NewTokenBucket(AuthenticatedRate)
	tb.SetRate(UnauthenticatedRate)

	if tb.Rate() != UnauthenticatedRate {
		t.Errorf("Rate() = %f; want %f", tb.Rate(), UnauthenticatedRate)
	}
}

func TestNewRateLimiter(t *testing.T) {
	t.Run("authenticated", func(t *testing.T) {
		rl := NewRateLimiter(true)
		if !rl.IsAuthenticated() {
			t.Error("IsAuthenticated() = false; want true")
		}
		expectedRate := AuthenticatedRate
		if rl.bucket.Rate() != expectedRate {
			t.Errorf("bucket rate = %f; want %f", rl.bucket.Rate(), expectedRate)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		rl := NewRateLimiter(false)
		if rl.IsAuthenticated() {
			t.Error("IsAuthenticated() = true; want false")
		}
		expectedRate := UnauthenticatedRate
		if rl.bucket.Rate() != expectedRate {
			t.Errorf("bucket rate = %f; want %f", rl.bucket.Rate(), expectedRate)
		}
	})
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(true)
	ctx := context.Background()

	err := rl.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}

func TestRateLimiter_FallbackToUnauthenticated(t *testing.T) {
	t.Run("fallback from authenticated", func(t *testing.T) {
		rl := NewRateLimiter(true)
		rl.FallbackToUnauthenticated()

		if rl.IsAuthenticated() {
			t.Error("IsAuthenticated() = true; want false after fallback")
		}
		if rl.bucket.Rate() != UnauthenticatedRate {
			t.Errorf("bucket rate = %f; want %f", rl.bucket.Rate(), UnauthenticatedRate)
		}
	})

	t.Run("fallback when already unauthenticated", func(t *testing.T) {
		rl := NewRateLimiter(false)
		rl.FallbackToUnauthenticated() // should be no-op

		if rl.IsAuthenticated() {
			t.Error("IsAuthenticated() = true; want false")
		}
		if rl.bucket.Rate() != UnauthenticatedRate {
			t.Errorf("bucket rate = %f; want %f", rl.bucket.Rate(), UnauthenticatedRate)
		}
	})

	t.Run("fallback only logs once", func(t *testing.T) {
		rl := NewRateLimiter(true)
		rl.FallbackToUnauthenticated()
		rl.FallbackToUnauthenticated() // second call should not panic or error
		if rl.IsAuthenticated() {
			t.Error("IsAuthenticated() = true; want false")
		}
	})
}

func TestRateLimiter_String(t *testing.T) {
	t.Run("authenticated string", func(t *testing.T) {
		rl := NewRateLimiter(true)
		got := rl.String()
		want := "authenticated (30 requests/minute)"
		if got != want {
			t.Errorf("String() = %q; want %q", got, want)
		}
	})

	t.Run("unauthenticated string", func(t *testing.T) {
		rl := NewRateLimiter(false)
		got := rl.String()
		want := "unauthenticated (10 requests/minute)"
		if got != want {
			t.Errorf("String() = %q; want %q", got, want)
		}
	})
}
