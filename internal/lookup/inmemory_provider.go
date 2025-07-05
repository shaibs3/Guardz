package lookup

import (
	"context"
	"github.com/shaibs3/Guardz/internal/db"
	"sync"
)

type InMemoryProvider struct {
	mu     sync.RWMutex
	paths  map[string]int64
	urls   map[int64][]string
	nextID int64
}

func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		paths:  make(map[string]int64),
		urls:   make(map[int64][]string),
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

func (m *InMemoryProvider) GetURLsByPath(ctx context.Context, path string) ([]db.URLRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.paths[path]
	if !ok {
		return nil, nil
	}
	urls := m.urls[id]
	records := make([]db.URLRecord, 0, len(urls))
	for i, url := range urls {
		records = append(records, db.URLRecord{
			ID:     int64(i + 1),
			PathID: id,
			URL:    url,
		})
	}
	return records, nil
}
