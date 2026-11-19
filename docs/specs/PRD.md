# PRD: linkedin-archive

## Summary

`linkedin-archive` turns LinkedIn data export archives (the `.zip` produced by
*Settings → Data privacy → Get a copy of your data*) into a searchable,
queryable database. Users import each new export as it arrives; the database
accumulates history across exports without duplicating records.

## Problem

LinkedIn's export is a zip of loosely structured CSV files. It is difficult to
answer simple questions with it:

- "Which messages mentioned X, and who sent them?"
- "When did I connect with this person, and what did we last discuss?"
- "Which posts did I save, and when?"

The web UI's message search is limited, conversations that were archived or
filtered to spam are easy to miss, and each export is a full snapshot that
cannot be diffed against earlier ones by hand.

## Users

- **Individual professionals** who want to search their own LinkedIn history
  locally and privately.
- **Developers** who want an importable Go library for parsing LinkedIn
  exports into structured records.

## Goals

1. Import a LinkedIn export `.zip` (or an extracted directory) with one
   command, without manual extraction or cleanup.
2. Re-importing the same export is a no-op; importing a newer export adds only
   new records and updates "last seen" tracking on existing ones.
3. Full-text search over messages across all folders (inbox, archive, spam),
   with filters for sender, folder and date range.
4. Structured access to connections, invitations and saved items.
5. Never lose data: files the importer does not model are preserved as raw
   rows.
6. Run locally on a single file database by default (SQLite); optionally run
   against PostgreSQL for hosted or shared deployments.

## Non-goals

- Scraping LinkedIn or calling LinkedIn APIs. The only input is the official
  export.
- Fetching the content of saved posts or shared URLs from LinkedIn.
- Multi-tenant hosting in the first release.

## Use cases

| ID | Use case |
|----|----------|
| UC-1 | Find a half-remembered inbound message, such as an offer to buy or finance private-company equity, by searching for terms across every folder. |
| UC-2 | List all messages from or to a person, in conversation order. |
| UC-3 | Import a newer export and see what was added since the previous import. |
| UC-4 | List saved posts/items with their URLs and saved dates. |
| UC-5 | Look up a connection's company, position and connection date. |

## Data availability

LinkedIn offers two export types:

- **Basic** (fast, ~10 minutes): profile, connections, messages,
  invitations, endorsements, positions and similar.
- **Complete** (up to 24 hours): Basic plus activity such as `Shares.csv`
  (your posts), `Comments.csv`, `Reactions.csv` and saved items.

Saved items are therefore only available from the Complete export, or from
a targeted request for that category. The importer must accept either export
type and must not assume any file is present.

## Privacy

- All processing is local; the tool makes no network calls.
- The repository contains no real export data. Tests use synthetic fixtures.
- Database files are excluded from version control by default.

## Success metrics

- A Basic export of ~5,000 messages imports in under 10 seconds on a laptop.
- Search returns results in under 200 ms for that dataset.
- Importing the same archive twice yields zero new rows.
