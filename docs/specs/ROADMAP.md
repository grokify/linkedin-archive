# Roadmap: linkedin-archive

Phase status is derived from the status of its RMIs.

## Phase 1: Import and message search

| RMI | Item | Status |
|-----|------|--------|
| RMI-LIARCHIVE-001 | Archive reader: zip/dir via `fs.FS`, header-driven CSV, preamble skip, date parsing | Done |
| RMI-LIARCHIVE-002 | Ent schema and store: SQLite and PostgreSQL DSNs, migrations | Done |
| RMI-LIARCHIVE-003 | Idempotent importer: imports, people, connections, conversations, messages, invitations, raw records | Done |
| RMI-LIARCHIVE-004 | Message full-text search: SQLite FTS5 and PostgreSQL implementations | Done |
| RMI-LIARCHIVE-005 | CLI: `import`, `search`, `imports`, `stats` | Done |

## Phase 2: Complete export coverage

| RMI | Item | Status |
|-----|------|--------|
| RMI-LIARCHIVE-006 | Saved items entity and `saved` command | Done |
| RMI-LIARCHIVE-007 | Verify saved-item column mapping against a Complete export | Planned |
| RMI-LIARCHIVE-008 | Posts (`Shares.csv`), comments and reactions entities with search | Planned |
| RMI-LIARCHIVE-009 | Endorsements, positions and profile entities | Planned |
| RMI-LIARCHIVE-010 | Import diff report: added and no-longer-present records between imports | Planned |

## Phase 3: Integrations

| RMI | Item | Status |
|-----|------|--------|
| RMI-LIARCHIVE-011 | MCP server exposing search and lookup tools | Planned |
| RMI-LIARCHIVE-012 | Connection and person search across all entities | Planned |
| RMI-LIARCHIVE-013 | Conversation view command (full thread output) | Planned |
| RMI-LIARCHIVE-014 | JSON Schema for exported record types | Planned |
| RMI-LIARCHIVE-015 | Release automation and changelog | Planned |
