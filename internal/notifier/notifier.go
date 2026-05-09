package notifier

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lucasmacori/sniffy/internal/detector"
)

// Notifier is the strategy interface for sending notifications
type Notifier interface {
	Name() string
	Notify(ctx context.Context, finding detector.Finding) error
	GetConfidenceThreshold() float64
}

// Registry holds and manages multiple notification strategies
type Registry struct {
	notifiers []Notifier
}

// NewRegistry creates a new notifier registry
func NewRegistry(notifiers ...Notifier) *Registry {
	return &Registry{
		notifiers: notifiers,
	}
}

// Notify sends a notification through all registered notifiers that meet their confidence threshold
func (r *Registry) Notify(ctx context.Context, finding detector.Finding) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(r.notifiers))

	for _, n := range r.notifiers {
		if finding.Confidence < n.GetConfidenceThreshold() {
			continue
		}

		wg.Add(1)
		go func(notifier Notifier) {
			defer wg.Done()
			if err := notifier.Notify(ctx, finding); err != nil {
				errChan <- fmt.Errorf("%s notifier failed: %w", notifier.Name(), err)
			}
		}(n)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}

	return nil
}

// Notifiers returns the list of registered notifiers
func (r *Registry) Notifiers() []Notifier {
	return r.notifiers
}

// RateLimitedNotifier wraps a notifier with rate limiting
type RateLimitedNotifier struct {
	notifier  Notifier
	limiter   <-chan time.Time
}

// NewRateLimitedNotifier creates a rate-limited notifier
func NewRateLimitedNotifier(notifier Notifier, ratePerSecond float64) *RateLimitedNotifier {
	interval := time.Second / time.Duration(ratePerSecond)
	return &RateLimitedNotifier{
		notifier: notifier,
		limiter:  time.Tick(interval),
	}
}

// Name returns the wrapped notifier's name
func (r *RateLimitedNotifier) Name() string {
	return fmt.Sprintf("ratelimited-%s", r.notifier.Name())
}

// Notify sends a notification with rate limiting
func (r *RateLimitedNotifier) Notify(ctx context.Context, finding detector.Finding) error {
	select {
	case <-r.limiter:
		return r.notifier.Notify(ctx, finding)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetConfidenceThreshold returns the wrapped notifier's threshold
func (r *RateLimitedNotifier) GetConfidenceThreshold() float64 {
	return r.notifier.GetConfidenceThreshold()
}

// BaseNotifier provides common notifier functionality
type BaseNotifier struct {
	name      string
	threshold float64
}

// Name returns the notifier name
func (b *BaseNotifier) Name() string {
	return b.name
}

// GetConfidenceThreshold returns the confidence threshold
func (b *BaseNotifier) GetConfidenceThreshold() float64 {
	return b.threshold
}
