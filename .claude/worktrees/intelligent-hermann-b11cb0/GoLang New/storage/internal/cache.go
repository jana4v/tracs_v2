package internal

import (
	"sync"
	"time"
)

// CacheEntry holds the last-written value for a mnemonic for change detection.
type CacheEntry struct {
	Value     string
	Timestamp time.Time
	IsBreak   bool
}

// Cache provides an in-memory cache of last-written values per mnemonic.
// Used by the storage rule engine for change detection (SRS 13.4).
type Cache struct {
	mu      sync.RWMutex
	entries map[string]CacheEntry
}

// NewCache creates a new empty cache.
func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]CacheEntry),
	}
}

// Get retrieves the cached entry for a mnemonic. Returns the entry and
// a boolean indicating whether it exists.
func (c *Cache) Get(mnemonic string) (CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[mnemonic]
	return entry, ok
}

// Update sets or updates the cached entry for a mnemonic.
func (c *Cache) Update(mnemonic string, value string, ts time.Time, isBreak bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[mnemonic] = CacheEntry{
		Value:     value,
		Timestamp: ts,
		IsBreak:   isBreak,
	}
}

// Delete removes a mnemonic from the cache.
func (c *Cache) Delete(mnemonic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, mnemonic)
}

// Len returns the number of entries in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
