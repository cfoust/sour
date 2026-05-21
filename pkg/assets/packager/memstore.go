package packager

import (
	"fmt"
	"sync"
)

// MemStore is an in-memory key-value store for asset data.
// It replaces the filesystem-based storage used by the disk Packager.
type MemStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemStore() *MemStore {
	return &MemStore{
		data: make(map[string][]byte),
	}
}

func (m *MemStore) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("not found: %s", key)
	}
	return data, nil
}

func (m *MemStore) Set(key string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
}

func (m *MemStore) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok
}

func (m *MemStore) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := 0
	for _, v := range m.data {
		total += len(v)
	}
	return total
}

func (m *MemStore) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}
