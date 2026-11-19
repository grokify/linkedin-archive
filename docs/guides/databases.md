# Databases

`linkedin-archive` stores data in SQLite by default and can use PostgreSQL
instead. The tables are the same in both. They are defined once with
[Ent](https://entgo.io) and created or updated automatically when the tool
opens the database.

## Choose the database

Every command accepts `--db`. You can also set the `LIARCHIVE_DB`
environment variable. The flag wins if both are set.

```bash
liarchive --db sqlite:$HOME/data/linkedin.db stats

export LIARCHIVE_DB=sqlite:$HOME/data/linkedin.db
liarchive stats
```

If neither is set, the database is `linkedin-archive.db` in the current
directory.

## SQLite

| Form | Meaning |
|------|---------|
| `sqlite:linkedin.db` | File relative to the current directory |
| `sqlite:///home/me/linkedin.db` | Absolute path |
| `sqlite::memory:` | In-memory database that disappears on exit (useful for tests) |

SQLite is the best choice for personal use:

- The whole archive is one file you can copy, back up or delete.
- No server to run.
- No C compiler needed. The driver is pure Go and includes full-text search.

The database uses write-ahead logging, so you may see `-wal` and `-shm` files
next to it while it is open.

## PostgreSQL

```bash
liarchive --db 'postgres://user:password@localhost:5432/linkedin?sslmode=disable' import export.zip
```

Both `postgres://` and `postgresql://` work. Create the database first; the
tables and the search index are created on first use.

Consider PostgreSQL when:

- The archive feeds a shared or hosted service.
- You want to join it with data that already lives in PostgreSQL.
- Several people's archives share one database. In that case you will want
  access controls such as row-level security, which you need to set up
  yourself.

!!! warning "Keep credentials out of shell history"
    Set `LIARCHIVE_DB` in your environment instead of typing a password on
    the command line.

## Moving between databases

There is no built-in copy between SQLite and PostgreSQL. Import the original
export files into the new database instead. Imports produce the same records
and keys, so the result matches.
