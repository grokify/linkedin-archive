package store

import (
	"context"
	"testing"
)

func TestParseDSN(t *testing.T) {
	cases := []struct {
		dsn, driver, conn string
		wantErr           bool
	}{
		{"sqlite:a.db", "sqlite", "file:a.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", false},
		{"sqlite:///tmp/a.db", "sqlite", "file:/tmp/a.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", false},
		{"sqlite::memory:", "sqlite", "file::memory:?_pragma=foreign_keys(1)", false},
		{"postgres://u@h/db", "pgx", "postgres://u@h/db", false},
		{"sqlite:", "", "", true},
		{"mysql://x", "", "", true},
	}
	for _, c := range cases {
		driver, _, conn, err := parseDSN(c.dsn)
		if (err != nil) != c.wantErr {
			t.Errorf("parseDSN(%q) err = %v", c.dsn, err)
			continue
		}
		if driver != c.driver || conn != c.conn {
			t.Errorf("parseDSN(%q) = %q, %q", c.dsn, driver, conn)
		}
	}
}

func TestOpenMemory(t *testing.T) {
	s, err := Open(context.Background(), "sqlite::memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
