# Installation

## Requirements

- Go 1.26 or later.
- No C compiler is needed. SQLite support uses a pure-Go driver.

## Install the command

```bash
go install github.com/grokify/linkedin-archive/cmd/liarchive@latest
```

To install a specific release:

```bash
go install github.com/grokify/linkedin-archive/cmd/liarchive@v0.1.0
```

Check that it works:

```bash
liarchive --help
```

## Use as a library

```bash
go get github.com/grokify/linkedin-archive
```

See [Go Library](library.md) for the packages and an example.

## Build from source

```bash
git clone https://github.com/grokify/linkedin-archive
cd linkedin-archive
go build ./cmd/liarchive
```

## Commands at a glance

| Command | What it does |
|---------|--------------|
| `liarchive import <zip or dir>...` | Import one or more exports |
| `liarchive search [query]` | Full-text search messages |
| `liarchive imports` | List the exports you have imported |
| `liarchive stats` | Show how many rows each table holds |
| `liarchive saved` | List saved items (Complete exports only) |

Every command accepts `--db` to choose the database. See
[Databases](databases.md).
