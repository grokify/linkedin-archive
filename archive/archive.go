// Package archive reads LinkedIn data export archives (zip files or
// extracted directories) into header-keyed CSV rows and typed records.
package archive

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
)

// Kind classifies an export.
type Kind string

const (
	KindBasic    Kind = "basic"
	KindComplete Kind = "complete"
	KindUnknown  Kind = "unknown"
)

// Archive is an opened LinkedIn export.
type Archive struct {
	fsys   fs.FS
	name   string
	sha256 string
	closer io.Closer
	files  []string // CSV paths within fsys, sorted
}

// Open opens a LinkedIn export from a .zip file or an extracted directory.
func Open(p string) (*Archive, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return New(os.DirFS(p), path.Base(p))
	}
	sum, err := hashFile(p)
	if err != nil {
		return nil, err
	}
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, fmt.Errorf("open zip %s: %w", p, err)
	}
	a, err := newArchive(zr, info.Name(), sum)
	if err != nil {
		if cerr := zr.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
		return nil, err
	}
	a.closer = zr
	return a, nil
}

// New creates an Archive over an arbitrary file system, such as an extracted
// export directory or an fstest.MapFS in tests. The content hash is computed
// from the CSV files' paths and bytes.
func New(fsys fs.FS, name string) (*Archive, error) {
	return newArchive(fsys, name, "")
}

func newArchive(fsys fs.FS, name, sum string) (*Archive, error) {
	a := &Archive{fsys: fsys, name: name}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), "__MACOSX") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(path.Ext(p), ".csv") && !strings.HasPrefix(d.Name(), "._") {
			a.files = append(a.files, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk archive: %w", err)
	}
	sort.Strings(a.files)
	if sum == "" {
		sum, err = a.hashFiles()
		if err != nil {
			return nil, err
		}
	}
	a.sha256 = sum
	return a, nil
}

// Close releases resources held by the archive.
func (a *Archive) Close() error {
	if a.closer != nil {
		return a.closer.Close()
	}
	return nil
}

// Name returns the archive's file or directory name.
func (a *Archive) Name() string { return a.name }

// SHA256 returns a hex content hash identifying the archive.
func (a *Archive) SHA256() string { return a.sha256 }

// Files returns the CSV file paths in the archive.
func (a *Archive) Files() []string { return append([]string(nil), a.files...) }

// Kind guesses whether this is a Basic or Complete export from the files present.
func (a *Archive) Kind() Kind {
	if len(a.files) == 0 {
		return KindUnknown
	}
	for _, f := range a.files {
		switch strings.ToLower(path.Base(f)) {
		case "shares.csv", "reactions.csv", "comments.csv":
			return KindComplete
		}
	}
	if a.Find("messages.csv") != "" || a.Find("connections.csv") != "" {
		return KindBasic
	}
	return KindUnknown
}

// Find returns the path of the first file whose base name matches name
// case-insensitively, or "" if none does.
func (a *Archive) Find(name string) string {
	for _, f := range a.files {
		if strings.EqualFold(path.Base(f), name) {
			return f
		}
	}
	return ""
}

// Match returns the paths of files whose lowercase base name satisfies fn.
func (a *Archive) Match(fn func(lowerBase string) bool) []string {
	var out []string
	for _, f := range a.files {
		if fn(strings.ToLower(path.Base(f))) {
			out = append(out, f)
		}
	}
	return out
}

func (a *Archive) hashFiles() (string, error) {
	h := sha256.New()
	for _, f := range a.files {
		b, err := fs.ReadFile(a.fsys, f)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := fmt.Fprintf(h, "%s\x00%d\x00", f, len(b)); err != nil {
			return "", err
		}
		if _, err := h.Write(b); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFile(p string) (sum string, err error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
