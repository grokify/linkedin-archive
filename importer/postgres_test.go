package importer_test

import (
	"context"
	"os"
	"testing"

	"github.com/grokify/linkedin-archive/store"
)

// TestImportAndSearchPostgres runs against a disposable PostgreSQL database
// named by LIARCHIVE_TEST_POSTGRES_DSN. The database must be empty.
func TestImportAndSearchPostgres(t *testing.T) {
	dsn := os.Getenv("LIARCHIVE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("LIARCHIVE_TEST_POSTGRES_DSN not set")
	}
	s, err := store.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	RunImportAndSearch(t, s)
}
