package importer_test

import (
	"io/fs"
	"testing"
	"time"
)

type archiveFS = fs.FS

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
