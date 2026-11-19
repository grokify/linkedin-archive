package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenZipAndDir(t *testing.T) {
	dir := t.TempDir()
	zpath := filepath.Join(dir, "export.zip")
	f, err := os.Create(zpath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("Basic_Export/messages.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(testMessages)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	a, err := Open(zpath)
	if err != nil {
		t.Fatal(err)
	}
	ms, err := a.Messages()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 4 || a.Name() != "export.zip" {
		t.Errorf("zip: %d messages, name %q", len(ms), a.Name())
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}

	edir := filepath.Join(dir, "extracted")
	if err := os.Mkdir(edir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(edir, "messages.csv"), []byte(testMessages), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := Open(edir)
	if err != nil {
		t.Fatal(err)
	}
	if d.Find("MESSAGES.CSV") == "" || d.SHA256() == "" {
		t.Errorf("dir archive: files %v", d.Files())
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(filepath.Join(dir, "missing.zip")); err == nil {
		t.Error("expected error for missing path")
	}
}
