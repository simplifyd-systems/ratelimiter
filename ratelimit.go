package ratelimiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	entries        map[string]*entry // Create a map to hold the rate limiters for each entry and a mutex.
	mu             sync.Mutex
	ratePerSec     rate.Limit
	burstPerPeriod int
}

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Run a background goroutine to remove old entries from the entries map.
// f = 1/T frequency = 1/Period
func New(cleanupInterval time.Duration, ratePerSec rate.Limit, burstPerPeriod int) *RateLimiter {
	rl := &RateLimiter{
		ratePerSec:     ratePerSec,
		burstPerPeriod: burstPerPeriod,
	}
	rl.entries = make(map[string]*entry)
	go rl.cleanupEntries(cleanupInterval)
	return rl
}

// Every minute check the map for entries that haven't been seen for
// more than duration and delete the entries.
func (rl *RateLimiter) cleanupEntries(duration time.Duration) {
	for {
		time.Sleep(duration)

		rl.mu.Lock()
		for k, v := range rl.entries {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.entries, k)
			}
		}
		rl.mu.Unlock()
	}
}

// Retrieve and return the rate limiter for the current entry if it
// already exists. Otherwise create a new rate limiter and add it to
// the entries map, using the k as the key.
func (rl *RateLimiter) getEntry(k string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.entries[k]
	if !exists {
		limiter := rate.NewLimiter(rl.ratePerSec, rl.burstPerPeriod)
		// Include the current time when creating a new entry.
		rl.entries[k] = &entry{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the entry.
	v.lastSeen = time.Now()
	return v.limiter
}

// Limit func
// returns true if we should limit, false otherwise
func (rl *RateLimiter) Limit(k string) bool {
	// Call the getEntry function to retreive the rate limiter for
	// the current entry.
	limiter := rl.getEntry(k)
	return limiter.Allow()
}

// This allows our callers remove entries for whatever reason their application
// or business logic dictates
func (rl *RateLimiter) RemoveEntry(k string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.entries, k)
}
