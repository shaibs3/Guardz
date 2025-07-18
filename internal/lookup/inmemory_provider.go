package lookup

import (
	"context"
	"sync"

	"github.com/shaibs3/Guardz/internal/db_model"
)

type InMemoryProvider struct {
	mu     sync.RWMutex
	paths  map[string]uint64
	urls   map[uint64][]string
	nextID uint64
}

func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		paths:  make(map[string]uint64),
		urls:   make(map[uint64][]string),
		nextID: 1,
	}
}

func (m *InMemoryProvider) StoreURLsForPath(ctx context.Context, path string, urls []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.paths[path]
	if !ok {
		id = m.nextID
		m.paths[path] = id
		m.nextID++
	}
	m.urls[id] = append([]string{}, urls...) // overwrite for idempotency
	return nil
}

func (m *InMemoryProvider) GetURLsByPath(ctx context.Context, path string) ([]db_model.URLRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.paths[path]
	if !ok {
		return nil, nil
	}
	urls := m.urls[id]
	records := make([]db_model.URLRecord, 0, len(urls))
	for i, url := range urls {
		records = append(records, db_model.URLRecord{
			ID:     uint64(i + 1), // #nosec G115
			PathID: id,
			URL:    url,
		})
	}
	return records, nil
}
