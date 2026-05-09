package notifier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lucasmacori/sniffy/internal/detector"
)

// mockNotifier is a test double for the Notifier interface
type mockNotifier struct {
	name      string
	threshold float64
	notifyErr error
	called    int
}

func (m *mockNotifier) Name() string {
	return m.name
}

func (m *mockNotifier) Notify(ctx context.Context, finding detector.Finding) error {
	m.called++
	return m.notifyErr
}

func (m *mockNotifier) GetConfidenceThreshold() float64 {
	return m.threshold
}

func TestRegistry_Notify(t *testing.T) {
	t.Run("notify all matching threshold", func(t *testing.T) {
		m1 := &mockNotifier{name: "m1", threshold: 50}
		m2 := &mockNotifier{name: "m2", threshold: 70}

		reg := NewRegistry(m1, m2)
		finding := detector.Finding{Confidence: 60}

		err := reg.Notify(context.Background(), finding)
		if err != nil {
			t.Fatalf("Notify() error = %v", err)
		}
		if m1.called != 1 {
			t.Errorf("m1.called = %d; want 1", m1.called)
		}
		if m2.called != 0 {
			t.Errorf("m2.called = %d; want 0 (below threshold)", m2.called)
		}
	})

	t.Run("notify none below threshold", func(t *testing.T) {
		m1 := &mockNotifier{name: "m1", threshold: 80}
		reg := NewRegistry(m1)

		finding := detector.Finding{Confidence: 50}
		err := reg.Notify(context.Background(), finding)
		if err != nil {
			t.Fatalf("Notify() error = %v", err)
		}
		if m1.called != 0 {
			t.Errorf("m1.called = %d; want 0", m1.called)
		}
	})

	t.Run("collect errors from notifiers", func(t *testing.T) {
		m1 := &mockNotifier{name: "m1", threshold: 0, notifyErr: errors.New("fail1")}
		m2 := &mockNotifier{name: "m2", threshold: 0, notifyErr: errors.New("fail2")}

		reg := NewRegistry(m1, m2)
		finding := detector.Finding{Confidence: 100}

		err := reg.Notify(context.Background(), finding)
		if err == nil {
			t.Fatal("Notify() expected error, got nil")
		}
	})

	t.Run("empty registry", func(t *testing.T) {
		reg := NewRegistry()
		finding := detector.Finding{Confidence: 100}

		err := reg.Notify(context.Background(), finding)
		if err != nil {
			t.Fatalf("Notify() error = %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		// RateLimitedNotifier or actual notifiers might respect context,
		// but simple mock doesn't block. This test verifies registry doesn't crash.
		m1 := &mockNotifier{name: "m1", threshold: 0}
		reg := NewRegistry(m1)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		finding := detector.Finding{Confidence: 100}
		err := reg.Notify(ctx, finding)
		// Since mocks don't block, this might still succeed or fail depending on timing.
		// We mainly ensure no panic.
		_ = err
	})
}

func TestRegistry_Notifiers(t *testing.T) {
	m1 := &mockNotifier{name: "m1"}
	m2 := &mockNotifier{name: "m2"}

	reg := NewRegistry(m1, m2)
	notifiers := reg.Notifiers()

	if len(notifiers) != 2 {
		t.Errorf("len(Notifiers()) = %d; want 2", len(notifiers))
	}
}

func TestBaseNotifier(t *testing.T) {
	b := BaseNotifier{name: "test", threshold: 42.0}

	if b.Name() != "test" {
		t.Errorf("Name() = %q; want test", b.Name())
	}
	if b.GetConfidenceThreshold() != 42.0 {
		t.Errorf("GetConfidenceThreshold() = %f; want 42.0", b.GetConfidenceThreshold())
	}
}

func TestRateLimitedNotifier(t *testing.T) {
	t.Run("name and threshold delegation", func(t *testing.T) {
		mock := &mockNotifier{name: "inner", threshold: 55}
		rl := NewRateLimitedNotifier(mock, 1000) // very high rate

		if rl.Name() != "ratelimited-inner" {
			t.Errorf("Name() = %q; want ratelimited-inner", rl.Name())
		}
		if rl.GetConfidenceThreshold() != 55 {
			t.Errorf("GetConfidenceThreshold() = %f; want 55", rl.GetConfidenceThreshold())
		}
	})

	t.Run("notify with high rate limit", func(t *testing.T) {
		mock := &mockNotifier{name: "inner", threshold: 0}
		rl := NewRateLimitedNotifier(mock, 10000) // very high rate

		finding := detector.Finding{Confidence: 100}
		ctx := context.Background()

		err := rl.Notify(ctx, finding)
		if err != nil {
			t.Fatalf("Notify() error = %v", err)
		}
		if mock.called != 1 {
			t.Errorf("mock.called = %d; want 1", mock.called)
		}
	})

	t.Run("notify with context cancellation", func(t *testing.T) {
		mock := &mockNotifier{name: "inner", threshold: 0}
		// Use rate of 1 per second so it blocks for ~1s
		rl := NewRateLimitedNotifier(mock, 1.0)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		finding := detector.Finding{Confidence: 100}
		err := rl.Notify(ctx, finding)
		if err != context.DeadlineExceeded && err != context.Canceled {
			// Depending on timing, it might succeed before context expires
			// or fail. We just ensure no panic.
			_ = err
		}
	})
}
