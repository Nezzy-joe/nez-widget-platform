package handlers

import (
	"strings"
	"sync"
	"time"
)

const maxTrackedRateLimitBuckets = 10_000

type rateLimitBucket struct {
	count   int
	resetAt time.Time
}

type rateLimitRule struct {
	key   string
	limit int
}

// SubmissionRateLimiter keeps independent limits for each client IP
// and public widget. It is intended for a single API instance.
type SubmissionRateLimiter struct {
	mu sync.Mutex

	buckets        map[string]rateLimitBucket
	perIPLimit     int
	perWidgetLimit int
	window         time.Duration
	maxBuckets     int
	now            func() time.Time
	lastCleanup    time.Time
}

// NewSubmissionRateLimiter creates a rate limiter.
// Invalid configuration values are replaced with safe defaults.
func NewSubmissionRateLimiter(
	perIPLimit int,
	perWidgetLimit int,
	window time.Duration,
) *SubmissionRateLimiter {
	if perIPLimit < 1 {
		perIPLimit = 20
	}
	if perWidgetLimit < 1 {
		perWidgetLimit = 100
	}
	if window <= 0 {
		window = time.Minute
	}

	return &SubmissionRateLimiter{
		buckets:        make(map[string]rateLimitBucket),
		perIPLimit:     perIPLimit,
		perWidgetLimit: perWidgetLimit,
		window:         window,
		maxBuckets:     maxTrackedRateLimitBuckets,
		now:            time.Now,
	}
}

// Allow atomically checks and consumes both the IP and widget limits.
// A denied request does not consume either counter.
func (l *SubmissionRateLimiter) Allow(
	clientIP string,
	publicWidgetID string,
) bool {
	if l == nil || strings.TrimSpace(publicWidgetID) == "" {
		return false
	}

	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		clientIP = "unknown"
	}

	now := l.now()

	rules := []rateLimitRule{
		{
			key:   "ip:" + clientIP,
			limit: l.perIPLimit,
		},
		{
			key:   "widget:" + publicWidgetID,
			limit: l.perWidgetLimit,
		},
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Periodically remove expired entries instead of scanning the map
	// on every request.
	if l.lastCleanup.IsZero() ||
		now.Sub(l.lastCleanup) >= l.window/2 {
		l.removeExpired(now)
		l.lastCleanup = now
	}

	for _, rule := range rules {
		bucket, exists := l.buckets[rule.key]
		if exists &&
			now.Before(bucket.resetAt) &&
			bucket.count >= rule.limit {
			return false
		}
	}

	// Reserve space for new buckets before incrementing either counter.
	needed := l.missingBuckets(rules, now)
	if len(l.buckets)+needed > l.maxBuckets {
		l.removeExpired(now)
		l.lastCleanup = now

		needed = l.missingBuckets(rules, now)
		if len(l.buckets)+needed > l.maxBuckets {
			// Fail closed rather than allow unlimited tracking growth.
			return false
		}
	}

	for _, rule := range rules {
		bucket, exists := l.buckets[rule.key]
		if !exists || !now.Before(bucket.resetAt) {
			bucket = rateLimitBucket{
				resetAt: now.Add(l.window),
			}
		}

		bucket.count++
		l.buckets[rule.key] = bucket
	}

	return true
}

func (l *SubmissionRateLimiter) missingBuckets(
	rules []rateLimitRule,
	now time.Time,
) int {
	missing := 0

	for _, rule := range rules {
		bucket, exists := l.buckets[rule.key]
		if !exists || !now.Before(bucket.resetAt) {
			missing++
		}
	}

	return missing
}

// removeExpired must be called while the mutex is held.
func (l *SubmissionRateLimiter) removeExpired(now time.Time) {
	for key, bucket := range l.buckets {
		if !now.Before(bucket.resetAt) {
			delete(l.buckets, key)
		}
	}
}
