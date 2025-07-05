package lookup

import (
	"context"

	"github.com/shaibs3/Guardz/internal/db"
)

type DbProvider interface {
	StoreURLsForPath(ctx context.Context, path string, urls []string) error
	GetURLsByPath(ctx context.Context, path string) ([]db.URLRecord, error)
}
