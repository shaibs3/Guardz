package lookup

import "context"

type DbProvider interface {
	Lookup(ctx context.Context, ip string) (city string, country string, err error)
}
