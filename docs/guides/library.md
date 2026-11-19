# Go Library

The `liarchive` command is a thin layer over packages you can use directly.

```bash
go get github.com/grokify/linkedin-archive
```

## Packages

| Package | Purpose |
|---------|---------|
| `archive` | Open an export (zip or folder); read CSV files by header name; typed records |
| `store` | Open SQLite or PostgreSQL by DSN and create or update the tables |
| `importer` | Import an archive into a store |
| `search` | Query and result types, and the query syntax parser |
| `search/engine` | Pick the right search implementation for a store |
| `search/sqlitefts`, `search/pgfts` | SQLite and PostgreSQL search implementations |
| `ent` | Ent client and schema, for your own queries |

## Import and search

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grokify/linkedin-archive/archive"
	"github.com/grokify/linkedin-archive/importer"
	"github.com/grokify/linkedin-archive/search"
	"github.com/grokify/linkedin-archive/search/engine"
	"github.com/grokify/linkedin-archive/store"
)

func main() {
	ctx := context.Background()

	s, err := store.Open(ctx, "sqlite:linkedin.db")
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	a, err := archive.Open("Basic_LinkedInDataExport_11-18-2026.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	res, err := importer.Import(ctx, s.Client, a)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("import #%d, %d new messages\n", res.ImportID, res.Stats["messages"].Inserted)

	srch, err := engine.New(ctx, s)
	if err != nil {
		log.Fatal(err)
	}
	results, err := srch.Search(ctx, search.Query{
		Text:   `acme (shares OR equity)`,
		Folder: "ARCHIVE",
		Limit:  10,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range results {
		fmt.Println(r.SentAt, r.From, r.Snippet)
	}
}
```

## Read an export without a database

The `archive` package works on its own, for example to convert an export to
another format:

```go
a, err := archive.Open("export.zip")
if err != nil {
	return err
}
defer a.Close()

msgs, err := a.Messages()        // []archive.Message
conns, err := a.Connections()    // []archive.Connection
invs, err := a.Invitations()     // []archive.Invitation
saved, err := a.SavedItems()     // []archive.SavedItem

// Any file, as rows keyed by header name
t, err := a.ReadCSV(a.Find("Skills.csv"))
for _, row := range t.Rows {
	fmt.Println(row.Get("Name"))
}
```

`archive.New` accepts any `fs.FS`, such as `fstest.MapFS` in tests.

Helpers:

| Function | Purpose |
|----------|---------|
| `archive.ParseTime(s)` | Parse any date format found in exports |
| `archive.HTMLToText(s)` | Strip HTML and collapse whitespace |
| `archive.NormalizeProfileURL(u)` | Normalize a LinkedIn profile URL |

## Query with Ent

`store.Store.Client` is a regular Ent client:

```go
import "github.com/grokify/linkedin-archive/ent/connection"

conns, err := s.Client.Connection.Query().
	Where(connection.CompanyContains("Acme")).
	All(ctx)
```

## Parse search queries

`search.Parse` turns the query syntax into terms, if you want to build your
own search on top:

```go
terms, err := search.Parse(`acme (shares OR "stock options") -recruiter`)
```

It returns `search.ErrOnlyNegated`, `search.ErrUnbalanced` or
`search.ErrNestingTooDeep` for queries it can't use.
