# Importing

## Import an export

```bash
liarchive import Basic_LinkedInDataExport_11-18-2026.zip
```

The output shows what was found in each file:

```text
Basic_LinkedInDataExport_11-18-2026.zip: import #1 (basic export) in 837ms
  ENTITY                   SEEN  NEW   SKIPPED
  connections              2911  2911  34
  conversations            1430  1430  0
  invitations              310   310   0
  messages                 4809  4809  0
  raw:Skills.csv           58    58    0
  ...
```

| Column | Meaning |
|--------|---------|
| `SEEN` | Rows read from this export |
| `NEW` | Rows not in the database before |
| `SKIPPED` | Rows with nothing to identify them by, such as connections from deactivated accounts |

Rows named `raw:<file>` come from files that don't have their own table. They
are stored in `raw_records` so nothing is lost.

You can import several exports at once, oldest first:

```bash
liarchive import exports/2025-06.zip exports/2026-11.zip
```

## Import again, or import a newer export

Imports are safe to repeat.

- **The same export again** is recognized by its content hash and skipped:

    ```text
    Basic_LinkedInDataExport_11-18-2026.zip: already imported (import #1)
    ```

- **A newer export** adds only records that weren't there before. Records
  already in the database are matched and updated, not duplicated.

Each record stores two import numbers:

| Field | Meaning |
|-------|---------|
| `first_seen_import_id` | The import where the record first appeared |
| `last_seen_import_id` | The most recent import that still contained it |

So after importing a newer export you can ask what is new and what has gone,
for example a connection who is no longer in your list. See
[Data Model and SQL](data-model.md#changes-between-imports).

A message that moved folders between exports, for example from Inbox to
Archive, keeps its identity and has its folder updated.

!!! note "Basic after Complete"
    Only files present in an export update "last seen". Importing a Basic
    export after a Complete one does not make your posts or saved items look
    removed.

## List imports

```bash
liarchive imports
```

```text
ID  IMPORTED                   KIND   SOURCE                                    NEW MESSAGES  NEW CONNECTIONS
1   2026-11-19T07:12:03-08:00  basic  Basic_LinkedInDataExport_11-18-2026.zip  4809          2911
```

## Count rows

```bash
liarchive stats
```

```text
imports        1
people         3426
connections    2911
conversations  1430
messages       4809
invitations    310
saved_items    0
raw_records    2556
```

## Saved items

```bash
liarchive saved
liarchive saved --json
```

This lists saved posts and items with their URL and the date you saved them.
They come only from a Complete export. See
[Exporting Your LinkedIn Data](exporting.md#basic-and-complete-exports).
