# Exporting Your LinkedIn Data

`linkedin-archive` works only from LinkedIn's official data export. It does
not log in to LinkedIn or collect data from the website.

## Request an export

1. On LinkedIn, open **Settings & Privacy**.
2. Go to **Data privacy** → **Get a copy of your data**.
3. Choose what to download (see below) and select **Request archive**.
4. LinkedIn emails you when the file is ready. Download the `.zip`.

You can import the `.zip` directly. There is no need to extract it first,
though an extracted folder works too.

## Basic and Complete exports

LinkedIn offers two kinds of export.

| | Basic | Complete |
|---|-------|----------|
| Typical wait | About 10 minutes | Up to 24 hours |
| Messages, connections, invitations | Yes | Yes |
| Profile, positions, skills, endorsements | Yes | Yes |
| Your posts, comments and reactions | No | Yes |
| Saved items (saved posts) | No | Yes |

`liarchive` accepts either one and detects which kind it is. The kind is
shown when you import and by `liarchive imports`.

!!! tip "Saved posts"
    Saved items appear only in a Complete export. You can also request just
    that category of data, which usually arrives much sooner than a full
    Complete export.

## What is in an export

An export is a set of CSV files. The ones `linkedin-archive` turns into
tables are:

| File | Table |
|------|-------|
| `messages.csv`, and other `*_messages.csv` files such as `learning_coach_messages.csv` | `messages`, `conversations`, `persons` |
| `Connections.csv` | `connections`, `persons` |
| `Invitations.csv` | `invitations` |
| Saved items or saved posts files | `saved_items` |
| Every other CSV file | `raw_records` (each row kept as JSON) |

## Known quirks

The importer handles these for you. They are listed here so the results make
sense.

- **Each export is a full snapshot.** It contains your whole history, not just
  what changed since the last one.
- **Messages have no ID.** The importer builds a stable one from each
  message's conversation, time, sender and content.
- **Some senders have no profile URL.** Sponsored InMail and "LinkedIn Member"
  messages are matched by sender name instead.
- **Message text is often HTML.** The original is kept, and a plain-text copy
  is used for search.
- **Connections from deactivated accounts** appear with only a connection
  date. They are counted as skipped because there is nothing to identify
  them by.
- **Dates come in many formats**, such as `2026-11-17 20:18:55 UTC`,
  `31 Oct 2026` and `10/30/26, 5:04 PM`. Times without a time zone are
  treated as UTC.
