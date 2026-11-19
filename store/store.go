// Package store opens a linkedin-archive database (SQLite or PostgreSQL)
// and runs schema migrations.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	_ "modernc.org/sqlite"             // registers the "sqlite" database/sql driver

	"github.com/grokify/linkedin-archive/ent"
)

// DefaultDSN is used when no DSN is configured.
const DefaultDSN = "sqlite:linkedin-archive.db"

// Store bundles the Ent client with the underlying database handle, which
// engine-specific packages (such as full-text search) need.
type Store struct {
	Client  *ent.Client
	DB      *sql.DB
	Dialect string // dialect.SQLite or dialect.Postgres
}

// Open opens the database identified by dsn and migrates the schema.
//
// Supported DSNs:
//
//	sqlite:path/to/file.db       (also sqlite://path, sqlite:///abs/path)
//	sqlite::memory:              in-memory database
//	postgres://user:pass@host/db (also postgresql://)
func Open(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		dsn = DefaultDSN
	}
	driverName, dia, conn, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driverName, conn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dia, err)
	}
	if dia == dialect.SQLite {
		// A single connection keeps in-memory databases shared and avoids
		// SQLITE_BUSY from concurrent writers.
		db.SetMaxOpenConns(1)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, closeWith(db, fmt.Errorf("connect %s: %w", dia, err))
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dia, db)))
	if err := client.Schema.Create(ctx); err != nil {
		return nil, closeWith(db, fmt.Errorf("migrate: %w", err))
	}
	return &Store{Client: client, DB: db, Dialect: dia}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.Client.Close()
}

func parseDSN(dsn string) (driverName, dia, conn string, err error) {
	switch {
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return "pgx", dialect.Postgres, dsn, nil
	case strings.HasPrefix(dsn, "sqlite:"):
		p := strings.TrimPrefix(dsn, "sqlite:")
		p = strings.TrimPrefix(p, "//")
		if p == "" {
			return "", "", "", errors.New("sqlite DSN has no path")
		}
		pragmas := "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
		if p == ":memory:" {
			return "sqlite", dialect.SQLite, "file::memory:?_pragma=foreign_keys(1)", nil
		}
		sep := "?"
		if strings.Contains(p, "?") {
			sep = "&"
		}
		return "sqlite", dialect.SQLite, "file:" + p + sep + pragmas, nil
	default:
		return "", "", "", fmt.Errorf("unsupported DSN %q: use sqlite:<path> or postgres://", dsn)
	}
}

func closeWith(db *sql.DB, err error) error {
	if cerr := db.Close(); cerr != nil {
		return errors.Join(err, cerr)
	}
	return err
}
