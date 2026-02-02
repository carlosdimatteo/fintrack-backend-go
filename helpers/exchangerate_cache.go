package helpers

import (
	"sync"
	"time"
)

// ExchangeRateCacheTTL is how long we keep a cached exchange-rate response before re-fetching.
const ExchangeRateCacheTTL = 12 * time.Hour

type exchangeRateEntry struct {
	body     []byte
	cachedAt time.Time
}

var exchangeRateCache = struct {
	mu      sync.RWMutex
	entries map[string]exchangeRateEntry
}{entries: make(map[string]exchangeRateEntry)}

// ExchangeRateCacheGet returns the cached response body and the time it was fetched (cachedAt)
// for the given pair (key = "from|to"). ok is true only if the entry exists and is still within ExchangeRateCacheTTL.
// cachedAt can be used to set last_fetched on the response so clients can verify cache behavior.
func ExchangeRateCacheGet(key string) (body []byte, cachedAt time.Time, ok bool) {
	now := time.Now()
	exchangeRateCache.mu.RLock()
	entry, exists := exchangeRateCache.entries[key]
	exchangeRateCache.mu.RUnlock()
	if !exists || now.Sub(entry.cachedAt) >= ExchangeRateCacheTTL {
		return nil, time.Time{}, false
	}
	return entry.body, entry.cachedAt, true
}

// ExchangeRateCacheSet stores the response body for the given pair (key = "from|to").
func ExchangeRateCacheSet(key string, body []byte) {
	exchangeRateCache.mu.Lock()
	exchangeRateCache.entries[key] = exchangeRateEntry{body: body, cachedAt: time.Now()}
	exchangeRateCache.mu.Unlock()
}
