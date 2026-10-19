package handlers

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubmissionRateLimiterEnforcesIPLimit(t *testing.T) {
	limiter := NewSubmissionRateLimiter(2, 10, time.Minute)

	if !limiter.Allow("192.0.2.1", "widget-a") {
		t.Fatal("first request should be allowed")
	}

	if !limiter.Allow("192.0.2.1", "widget-b") {
		t.Fatal("second request should be allowed")
	}

	if limiter.Allow("192.0.2.1", "widget-c") {
		t.Fatal("third request from the same IP should be denied")
	}
}

func TestSubmissionRateLimiterEnforcesWidgetLimit(t *testing.T) {
	limiter := NewSubmissionRateLimiter(10, 2, time.Minute)

	if !limiter.Allow("192.0.2.1", "widget-a") {
		t.Fatal("first request should be allowed")
	}

	if !limiter.Allow("192.0.2.2", "widget-a") {
		t.Fatal("second request should be allowed")
	}

	if limiter.Allow("192.0.2.3", "widget-a") {
		t.Fatal("third request for the same widget should be denied")
	}
}

func TestSubmissionRateLimiterResetsAfterWindow(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	window := time.Minute

	limiter := NewSubmissionRateLimiter(1, 10, window)
	limiter.now = func() time.Time {
		return now
	}

	if !limiter.Allow("192.0.2.1", "widget-a") {
		t.Fatal("first request should be allowed")
	}

	if limiter.Allow("192.0.2.1", "widget-a") {
		t.Fatal("second request should be denied within the window")
	}

	now = now.Add(window + time.Second)

	if !limiter.Allow("192.0.2.1", "widget-a") {
		t.Fatal("request should be allowed after the window expires")
	}
}

func TestSubmissionRateLimiterBoundsTrackedBuckets(t *testing.T) {
	limiter := NewSubmissionRateLimiter(100, 100, time.Minute)
	limiter.maxBuckets = 2

	if !limiter.Allow("192.0.2.1", "widget-a") {
		t.Fatal("first request should be allowed")
	}

	if limiter.Allow("192.0.2.2", "widget-b") {
		t.Fatal("new keys should be denied when capacity is exhausted")
	}

	if len(limiter.buckets) > limiter.maxBuckets {
		t.Fatalf(
			"tracked buckets exceeded limit: got %d, max %d",
			len(limiter.buckets),
			limiter.maxBuckets,
		)
	}
}

func TestSubmissionRateLimiterIsConcurrencySafe(t *testing.T) {
	const goroutines = 100
	const perIPLimit = 10

	limiter := NewSubmissionRateLimiter(
		perIPLimit,
		goroutines+1,
		time.Minute,
	)

	var allowed atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			widgetID := "widget-" + string(rune('A'+i))
			if limiter.Allow("192.0.2.1", widgetID) {
				allowed.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if got := allowed.Load(); got != perIPLimit {
		t.Fatalf("expected %d allowed requests, got %d", perIPLimit, got)
	}
}
