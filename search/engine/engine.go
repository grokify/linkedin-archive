// Package engine selects the message search implementation for a store's
// database dialect.
package engine

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect"

	"github.com/grokify/linkedin-archive/search"
	"github.com/grokify/linkedin-archive/search/pgfts"
	"github.com/grokify/linkedin-archive/search/sqlitefts"
	"github.com/grokify/linkedin-archive/store"
)

// New returns the Searcher for s, creating its index if needed.
func New(ctx context.Context, s *store.Store) (search.Searcher, error) {
	switch s.Dialect {
	case dialect.SQLite:
		return sqlitefts.New(ctx, s.DB)
	case dialect.Postgres:
		return pgfts.New(ctx, s.DB)
	default:
		return nil, fmt.Errorf("search: unsupported dialect %q", s.Dialect)
	}
}
