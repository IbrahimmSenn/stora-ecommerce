package cache

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	val       []byte
	expiresAt time.Time
}

// Memory is an in-process cache with per-entry TTLs. Expired entries are swept
// periodically so the map doesn't grow unbounded. This is the default backing
// when no Redis is configured.
type Memory struct {
	mu   sync.RWMutex
	data map[string]entry
}

// NewMemory builds an in-memory cache and starts a background sweep at the given
// interval (use a few seconds to a minute).
func NewMemory(sweep time.Duration) *Memory {
	m := &Memory{data: make(map[string]entry)}
	go m.sweepLoop(sweep)
	return m
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	e, ok := m.data[key]
	m.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false, nil
	}
	return e.val, true, nil
}

func (m *Memory) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	m.mu.Lock()
	m.data[key] = entry{val: val, expiresAt: time.Now().Add(ttl)}
	m.mu.Unlock()
	return nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
	return nil
}

func (m *Memory) sweepLoop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		m.mu.Lock()
		for k, e := range m.data {
			if now.After(e.expiresAt) {
				delete(m.data, k)
			}
		}
		m.mu.Unlock()
	}
}
