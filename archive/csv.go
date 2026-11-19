package archive

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// Row is a CSV record keyed by header name.
type Row map[string]string

// Get returns the value of the first matching header, compared
// case-insensitively, trimmed of surrounding whitespace.
func (r Row) Get(names ...string) string {
	for _, n := range names {
		if v, ok := r[n]; ok {
			return strings.TrimSpace(v)
		}
	}
	for _, n := range names {
		for k, v := range r {
			if strings.EqualFold(k, n) {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

// Table is a parsed CSV file.
type Table struct {
	Path   string
	Header []string
	Rows   []Row
}

// ReadCSV parses the CSV file at p. Lines before the header are skipped:
// the header is the first record whose fields include all of want (compared
// case-insensitively). If want is empty, the first record is the header,
// except that a leading "Notes:" block is skipped.
func (a *Archive) ReadCSV(p string, want ...string) (*Table, error) {
	b, err := fs.ReadFile(a.fsys, p)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	t, err := ParseCSV(b, want...)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	t.Path = p
	return t, nil
}

// ParseCSV parses CSV bytes as described in ReadCSV.
func ParseCSV(b []byte, want ...string) (*Table, error) {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	r := csv.NewReader(bytes.NewReader(b))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	recs, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	hi := headerIndex(recs, want)
	if hi < 0 {
		return &Table{}, nil
	}
	header := dedupeHeader(recs[hi])
	t := &Table{Header: header}
	for _, rec := range recs[hi+1:] {
		if isBlank(rec) {
			continue
		}
		row := make(Row, len(header))
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			} else {
				row[h] = ""
			}
		}
		t.Rows = append(t.Rows, row)
	}
	return t, nil
}

func headerIndex(recs [][]string, want []string) int {
	if len(want) > 0 {
		for i, rec := range recs {
			if containsAll(rec, want) {
				return i
			}
		}
		return -1
	}
	i := 0
	// Skip a "Notes:" preamble (as in Connections.csv) up to the first blank line.
	if len(recs) > 0 && len(recs[0]) == 1 && strings.EqualFold(strings.TrimSpace(recs[0][0]), "Notes:") {
		i = 1
		for i < len(recs) && len(recs[i]) == 1 {
			i++
		}
	}
	for i < len(recs) && isBlank(recs[i]) {
		i++
	}
	if i >= len(recs) {
		return -1
	}
	return i
}

func containsAll(rec, want []string) bool {
	for _, w := range want {
		found := false
		for _, f := range rec {
			if strings.EqualFold(strings.TrimSpace(f), w) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func dedupeHeader(h []string) []string {
	out := make([]string, len(h))
	seen := map[string]int{}
	for i, name := range h {
		name = strings.TrimSpace(name)
		if name == "" {
			name = "column_" + strconv.Itoa(i+1)
		}
		seen[name]++
		if n := seen[name]; n > 1 {
			name = name + "_" + strconv.Itoa(n)
		}
		out[i] = name
	}
	return out
}

func isBlank(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}
