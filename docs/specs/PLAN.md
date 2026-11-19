# Plan: linkedin-archive

## Approach

Build Phase 1 end to end against SQLite first, keeping engine-specific code in
`search/sqlitefts` and `search/pgfts` so the Ent model stays portable. Add
saved items early (Phase 2, RMI-LIARCHIVE-006) because the column mapping is
header-driven and tolerant, then confirm it against a real Complete export.

## Steps

1. **Module setup.** `go.mod` for `github.com/grokify/linkedin-archive`,
   dependencies at verified latest versions: `entgo.io/ent`,
   `github.com/spf13/cobra`, `modernc.org/sqlite`, `github.com/jackc/pgx/v5`.
2. **`archive` package.** `Open(path) (*Archive, error)` returning an
   `fs.FS`-backed archive; `ReadCSV(name)` returning header-keyed rows;
   `ParseTime`; typed readers `Messages()`, `Connections()`, `Invitations()`,
   `SavedItems()`, and `Files()` for raw iteration.
3. **Ent schema.** Entities listed in the TRD; generate with
   `go generate ./ent`.
4. **`store` package.** `Open(ctx, dsn) (*Store, error)` returning the Ent
   client, the `*sql.DB`, and the dialect; runs `client.Schema.Create`.
5. **`importer` package.** `Import(ctx, client, archive) (*Result, error)`
   implementing the semantics in the TRD.
6. **`search` packages.** Query parser, FTS5 translation, Postgres
   implementation, and `engine.New(ctx, store)` choosing by dialect.
7. **CLI.** Cobra commands over the library.
8. **Tests.** Synthetic `fstest.MapFS` exports; in-memory SQLite.
9. **Docs.** README with install, export instructions and examples.

## Risks

| Risk | Mitigation |
|------|------------|
| LinkedIn changes file or column names | Header-driven parsing; raw-record fallback preserves unknown data |
| Saved-item columns differ from the assumed headers | Unmapped columns kept in `extra`; RMI-LIARCHIVE-007 validates against a real export |
| Messages have no stable ID | Content-hash key with occurrence index |
| Timestamps without a zone (invitations) | Treated as UTC and documented |
| Personal data committed by accident | `.gitignore` covers `*.db`, `*.zip` and export directories; tests are synthetic |
