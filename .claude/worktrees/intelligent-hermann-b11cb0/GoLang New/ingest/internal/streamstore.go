package ingest

import (
	"sync"
	"time"
)

// StreamMeta describes a stream registered in the StreamStore.
type StreamMeta struct {
	ID        string // unique stream key: "TM1", "SMON1", "DTM", "UDTM"
	ChainType string // "TM", "SCOS", "programmatic"
	ChainName string // human-readable name (same as ID in most cases)
}

// StreamBuffer is an in-memory keyed value store for a single telemetry stream.
// Keys are parameter IDs only — e.g. "ACM05521" for TM, "cpu_mode" for SCOS.
// The mnemonic name is NOT part of the key; callers map id→mnemonic via the catalog.
// Thread-safe: all methods are safe for concurrent use.
type StreamBuffer struct {
	Meta     StreamMeta
	mu       sync.RWMutex
	values   map[string]string // id → value (current state)
	lastSent map[string]string // id → value (last dispatched, delta baseline)
	updated  time.Time
}

// Update merges the provided data into the buffer's current state.
func (b *StreamBuffer) Update(data map[string]string) {
	if len(data) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, v := range data {
		b.values[k] = v
	}
	b.updated = time.Now()
}

// Delta returns two maps: changed values (diff vs lastSent) and a full snapshot.
// Call CommitDelta after processing to advance the baseline.
func (b *StreamBuffer) Delta() (changed, all map[string]string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	changed = make(map[string]string, len(b.values))
	all = make(map[string]string, len(b.values))
	for k, v := range b.values {
		all[k] = v
		if last, ok := b.lastSent[k]; !ok || last != v {
			changed[k] = v
		}
	}
	return changed, all
}

// CommitDelta advances the lastSent baseline to the current values.
func (b *StreamBuffer) CommitDelta() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, v := range b.values {
		b.lastSent[k] = v
	}
}

// Snapshot returns a deep copy of the current values.
func (b *StreamBuffer) Snapshot() map[string]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	snap := make(map[string]string, len(b.values))
	for k, v := range b.values {
		snap[k] = v
	}
	return snap
}

// Len returns the number of values currently in the buffer.
func (b *StreamBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.values)
}

// StreamStore is a thread-safe registry of StreamBuffers, one per stream.
type StreamStore struct {
	mu      sync.RWMutex
	streams map[string]*StreamBuffer
	notify  chan struct{} // buffered(1); non-blocking write signals the dispatcher
}

// NewStreamStore creates an empty StreamStore.
func NewStreamStore() *StreamStore {
	return &StreamStore{
		streams: make(map[string]*StreamBuffer),
		notify:  make(chan struct{}, 1),
	}
}

// GetOrCreate returns the existing StreamBuffer for meta.ID, or creates and
// registers a new one if it does not exist yet.
func (s *StreamStore) GetOrCreate(meta StreamMeta) *StreamBuffer {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.streams[meta.ID]; ok {
		return b
	}
	b := &StreamBuffer{
		Meta:     meta,
		values:   make(map[string]string),
		lastSent: make(map[string]string),
	}
	s.streams[meta.ID] = b
	return b
}

// Get returns the StreamBuffer for id, or (nil, false) if not found.
func (s *StreamStore) Get(id string) (*StreamBuffer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.streams[id]
	return b, ok
}

// All returns a snapshot slice of all registered StreamBuffers.
func (s *StreamStore) All() []*StreamBuffer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]*StreamBuffer, 0, len(s.streams))
	for _, b := range s.streams {
		all = append(all, b)
	}
	return all
}

// Notify signals the dispatcher that new data is available.
// Non-blocking: if a signal is already pending, this is a no-op.
func (s *StreamStore) Notify() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

// NotifyCh returns the read-only channel the dispatcher listens on.
func (s *StreamStore) NotifyCh() <-chan struct{} {
	return s.notify
}
