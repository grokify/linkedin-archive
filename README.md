# LinkedIn Archive

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
![Coverage](https://img.shields.io/badge/coverage-70.6%25-yellowgreen)
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/grokify/linkedin-archive/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/grokify/linkedin-archive/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/grokify/linkedin-archive/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/grokify/linkedin-archive/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/grokify/linkedin-archive/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/grokify/linkedin-archive/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/linkedin-archive
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/linkedin-archive
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fh5pkit
 [loc-svg]: https://tokei.rs/b1/github/grokify/linkedin-archive
 [repo-url]: https://github.com/grokify/linkedin-archive
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/linkedin-archive/blob/master/LICENSE

Turn LinkedIn data export archives into a searchable database.

`linkedin-archive` reads the `.zip` that LinkedIn produces under
*Settings → Data privacy → Get a copy of your data* (or the extracted
directory) and loads it into SQLite or PostgreSQL. Import each new export as
it arrives: re-importing is a no-op and newer exports add only new records.

- Full-text search over all messages, including archived and spam folders
- Connections, conversations, invitations and saved items as typed tables
- Every other CSV in the export preserved as raw rows, so nothing is lost
- Library-first: the CLI is a thin layer over importable Go packages
- Runs locally with no network access; SQLite by default, PostgreSQL optional

## Install

```bash
go install github.com/grokify/linkedin-archive/cmd/liarchive@latest
```

## Usage

```bash
# Import an export (zip or extracted directory). Safe to repeat.
liarchive import Basic_LinkedInDataExport_11-18-2026.zip

# Search messages
liarchive search 'acme (shares OR equity OR "stock options")'
liarchive search 'tender offer' --folder ARCHIVE --since 2025-01-01
liarchive search --from "Jane Doe" --full

# Inspect
liarchive imports
liarchive stats
liarchive saved
```

The database defaults to `linkedin-archive.db` in the current directory.
Set `--db` or `LIARCHIVE_DB` to change it:

```bash
liarchive --db sqlite:$HOME/data/linkedin.db stats
liarchive --db postgres://user:pass@localhost/linkedin import export.zip
```

### Search syntax

| Syntax | Meaning |
|--------|---------|
| `a b` | both terms |
| `a OR b` | either term (AND binds tighter than OR) |
| `"a b"` | exact phrase |
| `-a` | exclude a term |
| `pre*` | prefix match |
| `(a OR b) c` | grouping |

English stemming is applied, so `shares` also matches `share`.

## Basic vs Complete exports

LinkedIn offers two export types. The **Basic** export (about 10 minutes)
includes messages, connections, invitations, profile and similar data. The
**Complete** export (up to 24 hours) adds activity such as your posts,
comments, reactions and saved items. `linkedin-archive` accepts either; saved
items appear only after importing a Complete export.

## Library

```go
a, err := archive.Open("export.zip")
// handle err
defer a.Close()

s, err := store.Open(ctx, "sqlite:linkedin.db")
// handle err
defer s.Close()

res, err := importer.Import(ctx, s.Client, a)
// handle err

srch, err := engine.New(ctx, s)
// handle err
results, err := srch.Search(ctx, search.Query{Text: "shares OR equity", Folder: "INBOX"})
```

| Package | Purpose |
|---------|---------|
| `archive` | Read exports; header-driven CSV parsing; typed records |
| `ent` | Ent schema and generated client |
| `store` | Open SQLite/PostgreSQL by DSN and migrate |
| `importer` | Idempotent import |
| `search` | Engine-neutral query types and syntax |
| `search/sqlitefts`, `search/pgfts` | Engine-specific full-text search |
| `search/engine` | Pick the search implementation for a store |

## Privacy

Exports contain personal data about you and your contacts. Keep databases and
exports out of version control (the included `.gitignore` excludes `*.db`,
`*.zip` and export directories).

## Development

```bash
go generate ./ent   # after changing ent/schema
go test ./...
golangci-lint run
```

PostgreSQL tests run when `LIARCHIVE_TEST_POSTGRES_DSN` names an empty,
disposable database.

Specs: [PRD](docs/specs/PRD.md) · [TRD](docs/specs/TRD.md) ·
[Plan](docs/specs/PLAN.md) · [Roadmap](docs/specs/ROADMAP.md)

## License

MIT
