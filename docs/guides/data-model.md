# Data Model and SQL

You can query the database directly with SQL, for example with
`sqlite3 linkedin-archive.db` or `psql`. The examples on this page are
SQLite; most work unchanged in PostgreSQL.

## Tables

| Table | One row per | Key columns |
|-------|-------------|-------------|
| `import_runs` | Imported export | `sha256`, `source_name`, `export_kind`, `imported_at`, `stats` |
| `persons` | Person seen anywhere in the archive | `key`, `profile_url`, `name` |
| `connections` | First-degree connection | `first_name`, `last_name`, `company`, `position`, `email`, `connected_on`, `person_id` |
| `conversations` | Message thread | `conversation_id` (LinkedIn's ID), `title` |
| `messages` | Message | `sent_at`, `from_name`, `sender_url`, `to_names`, `subject`, `body_text`, `content`, `folder`, `conversation_id`, `sender_id` |
| `message_recipients` | Message–recipient pair | `message_id`, `person_id` |
| `invitations` | Connection invitation | `from_name`, `to_name`, `sent_at`, `direction`, `message` |
| `saved_items` | Saved post or item | `url`, `saved_at`, `title`, `author`, `extra` |
| `raw_records` | Row of any other export file | `file`, `data` (JSON) |

`connections`, `conversations`, `messages`, `invitations`, `saved_items` and
`raw_records` also have `first_seen_import_id` and `last_seen_import_id`.

Notes on columns:

- `persons.key` is the person's normalized profile URL, or `name:<name>` when
  there is no URL.
- `messages.content` is the original text, which may be HTML.
  `messages.body_text` is the plain-text version used for search.
- `messages.conversation_id` and `messages.sender_id` refer to the `id`
  column of `conversations` and `persons`. LinkedIn's own conversation ID is
  `conversations.conversation_id`.
- `messages.to_names` is copied as-is from the export. Use
  `message_recipients` for reliable recipients, because names can contain
  commas.
- Times are stored in UTC.

## Example queries

### Who messages you most

```sql
SELECT p.name, p.profile_url, count(*) AS messages
FROM messages m
JOIN persons p ON p.id = m.sender_id
GROUP BY p.id
ORDER BY messages DESC
LIMIT 20;
```

### Connections by company

```sql
SELECT company, count(*) AS people
FROM connections
WHERE company <> ''
GROUP BY company
ORDER BY people DESC
LIMIT 20;
```

### Read a whole conversation

Use the `conversation:` value from a search result:

```sql
SELECT m.sent_at, m.from_name, m.body_text
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
WHERE c.conversation_id = '2-YTQ3MjU0ZDkt...'
ORDER BY m.sent_at;
```

### Full-text search in SQL (SQLite)

```sql
SELECT m.sent_at, m.from_name, snippet(message_fts, 1, '[', ']', '…', 20)
FROM message_fts
JOIN messages m ON m.id = message_fts.rowid
WHERE message_fts MATCH '"tender" AND "offer"'
ORDER BY bm25(message_fts);
```

### Read files that don't have their own table

Each row is stored as a JSON object keyed by the CSV header:

```sql
SELECT json_extract(data, '$.Name') AS skill
FROM raw_records
WHERE file = 'Skills.csv';
```

In PostgreSQL, use `data ->> 'Name'` instead of `json_extract`.

## Changes between imports

New since the latest import:

```sql
SELECT sent_at, from_name, substr(body_text, 1, 80)
FROM messages
WHERE first_seen_import_id = (SELECT max(id) FROM import_runs)
ORDER BY sent_at;
```

Connections that were in an earlier export but not the latest:

```sql
SELECT first_name, last_name, company
FROM connections
WHERE last_seen_import_id < (SELECT max(id) FROM import_runs);
```

## Import statistics

`import_runs.stats` holds the per-file counts shown when you import:

```sql
SELECT id, source_name, json_extract(stats, '$.messages.inserted') AS new_messages
FROM import_runs;
```
