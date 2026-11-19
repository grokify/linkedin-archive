# Searching Messages

`liarchive search` runs a full-text search over the subject, text and sender
name of every message, in every folder.

```bash
liarchive search 'tender offer'
```

```text
2026-10-01 09:00  Tender Desk  [INBOX]  https://www.linkedin.com/in/tender-desk
  A [tender] [offer] for Example Corp shares is open.
  conversation: 2-YTQ3MjU0ZDkt...

1 result(s)
```

Matching words are shown in `[brackets]`. Results are sorted by how well they
match.

## Query syntax

| Syntax | Meaning | Example |
|--------|---------|---------|
| `a b` | Both words | `stock options` |
| `a OR b` | Either word | `shares OR equity` |
| `"a b"` | Exact phrase | `"secondary sale"` |
| `-a` | Exclude a word | `ipo -recruiter` |
| `pre*` | Words starting with `pre` | `invest*` |
| `(a OR b) c` | Grouping | `acme (shares OR equity)` |

Rules to keep in mind:

- `AND` binds tighter than `OR`. `a b OR c` means `(a AND b) OR c`. Use
  parentheses when you mean `a AND (b OR c)`.
- `OR` must be in capitals. A lowercase `or` is searched as a word.
- An excluded word applies to the group it is in. `(a OR b) -c` finds
  messages with `a` or `b` that don't mention `c`.
- A query can't consist only of excluded words.
- Searches ignore case. SQLite also ignores accents, so `resume` matches
  `résumé`; PostgreSQL does not.

!!! info "Word stemming"
    English words are reduced to their root, so `shares` also matches `share`,
    `shared` and `sharing`. This finds more variations, but can also match
    words used in a different sense. Narrow the results with more words, a
    phrase, or an excluded word.

## Filters

Filters can be combined with a query, or used on their own to list messages.

| Flag | Meaning |
|------|---------|
| `--from <text>` | Sender name or profile URL contains this text |
| `--folder <name>` | Only this folder: `INBOX`, `ARCHIVE` or `SPAM` (any case) |
| `--since YYYY-MM-DD` | Sent on or after this date |
| `--until YYYY-MM-DD` | Sent before this date |
| `-n`, `--limit <n>` | At most this many results (default 50) |

Without a query, results are sorted newest first.

```bash
# Everything one person sent you
liarchive search --from "Jane Doe"

# Archived messages from 2025 that mention a phrase
liarchive search '"stock options"' --folder archive --since 2025-01-01 --until 2026-01-01
```

## Output

| Flag | Output |
|------|--------|
| *(none)* | Readable list with a snippet around the matching words |
| `--full` | The full plain text of each message instead of a snippet |
| `--json` | JSON, for scripts and other tools |

```bash
liarchive search 'shares OR equity' --json | jq '.[] | {sentAt, from, folder}'
```

Each JSON result includes `messageId`, `key`, `conversationId`,
`conversationTitle`, `sentAt`, `from`, `senderUrl`, `to`, `folder`,
`subject`, `snippet` and `rank`.

## Example: finding an offer you half-remember

Say you recall someone offering to buy, or lend against, your shares in a
private company, but not who or when:

```bash
liarchive search 'acme (shares OR equity OR stock OR liquidity OR secondary OR tender OR investor*)'
```

Then narrow down:

```bash
# Drop recruiting messages that mention equity
liarchive search 'acme (shares OR equity) -hiring -role -recruit*'

# Read the full text of the best matches
liarchive search 'acme liquidity' --full -n 5
```

To read the whole conversation for a result, use its conversation ID with SQL.
See [Data Model and SQL](data-model.md#read-a-whole-conversation).

## How search works

| Database | Engine |
|----------|--------|
| SQLite | FTS5 index with English stemming, kept up to date automatically as messages are imported |
| PostgreSQL | Built-in full-text search with the `english` configuration and a GIN index |

The index is created the first time you import or search. Both databases
accept the same query syntax.
