# LinkedIn Archive

Turn LinkedIn data export archives into a searchable database.

`linkedin-archive` reads the `.zip` file LinkedIn gives you when you request a
copy of your data, or the folder you extracted it to, and loads it into SQLite
or PostgreSQL. Import each new export as it arrives: importing the same
export again changes nothing, and a newer export adds only what is new.

```bash
go install github.com/grokify/linkedin-archive/cmd/liarchive@latest

liarchive import Basic_LinkedInDataExport_11-18-2026.zip
liarchive search 'acme (shares OR equity OR "stock options")'
```

## Features

- **Full-text search over every message**, including the Archive and Spam
  folders that are easy to miss in LinkedIn's own interface.
- **Repeat imports are safe.** Each record remembers the first and latest
  export it appeared in, so you can see what is new and what has disappeared.
- **Nothing is lost.** Messages, connections, conversations, invitations and
  saved items get their own tables. Every other file in the export is kept
  row by row.
- **Local and private.** The tool makes no network calls. SQLite is the
  default, so your data stays in one file on your machine.
- **PostgreSQL when you need it**, through the same `--db` flag.
- **Library first.** The `liarchive` command is a thin layer over Go packages
  you can import into your own programs.

## Where to start

| If you want to… | Read |
|-----------------|------|
| Install the command | [Installation](guides/installation.md) |
| Get your data from LinkedIn | [Exporting Your LinkedIn Data](guides/exporting.md) |
| Load an export, or a newer one | [Importing](guides/importing.md) |
| Find a message | [Searching Messages](guides/searching.md) |
| Use PostgreSQL or change the database file | [Databases](guides/databases.md) |
| Query the tables with SQL | [Data Model and SQL](guides/data-model.md) |
| Use the Go packages | [Go Library](guides/library.md) |

## Privacy

An export contains personal data about you and the people you have talked
to. Keep databases and exports out of version control. The repository's
`.gitignore` already excludes `*.db`, `*.zip` and export folders.
