# TRD: linkedin-archive

## Architecture

Library-first: all logic lives in importable packages; the CLI is a thin
adapter.

```
cmd/liarchive        Cobra CLI (import, search, stats, imports, saved)
archive/             Read export zip/dir; header-driven CSV parsing; typed records
ent/schema/          Ent schemas (source of truth for tables)
ent/                 Generated Ent code
store/               Open database by DSN (sqlite / postgres), run migrations
importer/            Idempotent import service
search/              Searcher interface and query syntax
search/sqlitefts/    SQLite FTS5 implementation
search/pgfts/        PostgreSQL full-text implementation
```

## Database support

| Engine | Driver | Use |
|--------|--------|-----|
| SQLite | `modernc.org/sqlite` (pure Go, FTS5 built in) | Default, local single file |
| PostgreSQL | `github.com/jackc/pgx/v5/stdlib` | Hosted or shared deployments |

Ent defines all tables and runs migrations for both engines. Full-text search
is engine-specific and lives in separate packages behind the
`search.Searcher` interface:

- **SQLite:** an FTS5 external-content table `message_fts` over
  `messages(subject, body_text, from_name)`, rebuilt after each import.
- **PostgreSQL:** a GIN expression index on
  `to_tsvector('simple', subject || ' ' || body_text || ' ' || from_name)`,
  queried with `websearch_to_tsquery`.

DSN format:

- `sqlite:///path/to/file.db` or `sqlite:file.db`
- `postgres://user:pass@host/db?sslmode=disable`

## Input handling

- Input is a `.zip` file or an extracted directory; both are read through
  `fs.FS`, so no extraction step is needed.
- CSV files are parsed by **header name**, not column position. Missing
  columns yield empty values; extra columns are kept.
- Preamble lines before the header (e.g. `Connections.csv` starts with a
  `Notes:` block) are skipped by locating the first line containing the
  expected header fields.
- A UTF-8 BOM is stripped.
- Duplicate header names (e.g. `Ad_Targeting.csv`) are suffixed (`_2`, `_3`).
- File names are matched case-insensitively by base name, so files in
  subdirectories and minor naming changes are tolerated.

### Date parsing

Exports use many formats. A shared parser tries, in order:

| Example | Source |
|---------|--------|
| `2026-11-17 20:18:55 UTC` | messages |
| `2025/10/08 11:50:14 UTC` | endorsements |
| `2026-04-26 14:07 UTC` | learning |
| `31 Oct 2026` | connections |
| `10/30/26, 5:04 PM` | invitations, saved jobs (no zone; treated as UTC) |
| `Wed Nov 18 20:32:07 GMT 2026` | company follows |
| `Jul 2023` | positions |
| RFC 3339 | fallback |

`N/A` and empty strings parse to "no value".

## Data model (Ent)

All imported entities carry `first_seen_import_id` and
`last_seen_import_id` edges/fields to support snapshot diffing.

| Entity | Natural key | Notes |
|--------|-------------|-------|
| `Import` | `sha256` of the archive bytes (zip) or sorted file hashes (dir) | `source_name`, `export_kind` (basic/complete/unknown), `imported_at`, `files` (JSON) |
| `Person` | `key` = normalized profile URL, else `name:<lowercased name>` | `profile_url`, `name` |
| `Connection` | person | `first_name`, `last_name`, `email`, `company`, `position`, `connected_on` |
| `Conversation` | LinkedIn `CONVERSATION ID` | `title`, `source_file` |
| `Message` | `key` = SHA-256 of (conversation id, date, sender url, from, subject, content) plus an occurrence index for exact repeats within one export | `sent_at`, `from_name`, `to_names`, `recipient_urls`, `subject`, `content` (raw), `body_text` (HTML stripped), `folder`, `attachments`, `source_file`; edges: conversation, sender, recipients |
| `Invitation` | SHA-256 of row | `from_name`, `to_name`, `sent_at`, `message`, `direction`, `inviter_url`, `invitee_url` |
| `SavedItem` | SHA-256 of (url, saved at) | `url`, `saved_at`, `title`, `author`, `content`, `extra` (JSON for other columns) |
| `RawRecord` | (`file`, `row_hash`) | `data` (JSON map of header → value) for files not modeled above |

### Message parsing details

- Message files: `messages.csv` plus any `*_messages.csv` with the same header
  (e.g. `learning_coach_messages.csv`); `source_file` records which.
- Recipients come from `RECIPIENT PROFILE URLS` (comma separated). The `TO`
  name list cannot be split reliably because names can contain commas, so it
  is stored verbatim.
- Senders without a profile URL (sponsored InMail, "LinkedIn Member") are
  keyed by name.
- `body_text` strips HTML tags, unescapes entities and collapses whitespace.

### Saved items

The importer recognizes any file whose base name matches `saved*item*` or
`saved*post*` (case-insensitive). It maps URL-like and date-like columns by
header name (`Saved Item`, `Url`, `Link`, `Saved At`, `Saved Date`,
`Created Time`, …) and keeps all other columns in `extra`. Column names are
confirmed against a Complete export before release.

## Import semantics

1. Hash the input. If an `Import` with that hash exists, report "already
   imported" and stop.
2. Create the `Import` row.
3. In one transaction per file, upsert each record by natural key:
   - New: insert with `first_seen = last_seen = this import`.
   - Existing: update `last_seen` (and mutable fields such as folder).
4. Only files present in this export update `last_seen`, so a Basic export
   does not mark Complete-only records as removed.
5. Rebuild the search index.
6. Store per-file counts (`seen`, `inserted`) on the `Import`.

## Search

`search.Query`:

| Field | Meaning |
|-------|---------|
| `Text` | Query string |
| `From` | Sender name or profile URL substring |
| `Folder` | `INBOX`, `ARCHIVE`, `SPAM`, … (empty = all) |
| `Since`, `Until` | Date range |
| `Limit` | Max results (default 50) |

Query syntax (portable across engines): bare words are ANDed, `"quoted
phrases"`, `OR`, `-excluded`, and `prefix*`. The SQLite implementation
translates it to FTS5; PostgreSQL uses `websearch_to_tsquery` (prefix
matching is SQLite-only).

Results include the message, conversation id/title, sender, a highlighted
snippet and a rank.

## CLI

```
liarchive import <zip|dir>        [--db DSN]
liarchive search <query>          [--from --folder --since --until --limit --json]
liarchive imports                 list imports with counts
liarchive stats                   row counts per entity
liarchive saved                   list saved items
```

`--db` defaults to `$LIARCHIVE_DB`, else `sqlite:linkedin-archive.db`.

## Testing

- Unit tests for CSV parsing, date parsing, HTML stripping, query translation
  and key generation.
- Importer tests against an in-memory SQLite database using a synthetic export
  built in the test (`fstest.MapFS`).
- PostgreSQL tests run only when `LIARCHIVE_TEST_POSTGRES_DSN` is set.
