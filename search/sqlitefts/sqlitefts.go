// Package sqlitefts implements message search with SQLite FTS5.
//
// It maintains an external-content FTS5 table, message_fts, over the
// messages table, kept in sync by triggers. The table and triggers are
// created (and the index built) on first use.
package sqlitefts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/grokify/linkedin-archive/search"
)

// Searcher searches messages in a SQLite database.
type Searcher struct {
	db *sql.DB
}

var setupSQL = []string{
	`CREATE VIRTUAL TABLE message_fts USING fts5(
		subject, body_text, from_name,
		content='messages', content_rowid='id',
		tokenize='porter unicode61 remove_diacritics 2')`,
	`CREATE TRIGGER message_fts_ai AFTER INSERT ON messages BEGIN
		INSERT INTO message_fts(rowid, subject, body_text, from_name)
		VALUES (new.id, new.subject, new.body_text, new.from_name);
	END`,
	`CREATE TRIGGER message_fts_ad AFTER DELETE ON messages BEGIN
		INSERT INTO message_fts(message_fts, rowid, subject, body_text, from_name)
		VALUES ('delete', old.id, old.subject, old.body_text, old.from_name);
	END`,
	`CREATE TRIGGER message_fts_au AFTER UPDATE OF subject, body_text, from_name ON messages BEGIN
		INSERT INTO message_fts(message_fts, rowid, subject, body_text, from_name)
		VALUES ('delete', old.id, old.subject, old.body_text, old.from_name);
		INSERT INTO message_fts(rowid, subject, body_text, from_name)
		VALUES (new.id, new.subject, new.body_text, new.from_name);
	END`,
	`INSERT INTO message_fts(message_fts) VALUES ('rebuild')`,
}

// New returns a Searcher, creating the FTS index if it does not exist.
func New(ctx context.Context, db *sql.DB) (*Searcher, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'message_fts'`).Scan(&n)
	if err != nil {
		return nil, fmt.Errorf("check fts table: %w", err)
	}
	if n == 0 {
		if err := setup(ctx, db); err != nil {
			return nil, err
		}
	}
	return &Searcher{db: db}, nil
}

func setup(ctx context.Context, db *sql.DB) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				err = fmt.Errorf("%w (rollback: %v)", err, rerr)
			}
		}
	}()
	for _, stmt := range setupSQL {
		if _, err = tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create fts index: %w", err)
		}
	}
	return tx.Commit()
}

// Rebuild rebuilds the full-text index from the messages table.
func (s *Searcher) Rebuild(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO message_fts(message_fts) VALUES ('rebuild')`)
	return err
}

// Search implements search.Searcher.
func (s *Searcher) Search(ctx context.Context, q search.Query) ([]search.Result, error) {
	terms, err := search.Parse(q.Text)
	if err != nil {
		return nil, err
	}
	var (
		where []string
		args  []any
		b     strings.Builder
	)
	if len(terms) > 0 {
		b.WriteString(`SELECT m.id, m.key, m.sent_at, m.from_name, coalesce(m.sender_url, ''),
			coalesce(m.to_names, ''), coalesce(m.folder, ''), m.subject,
			snippet(message_fts, 1, '[', ']', '…', 24), bm25(message_fts),
			coalesce(c.conversation_id, ''), coalesce(c.title, '')
		FROM message_fts
		JOIN messages m ON m.id = message_fts.rowid
		LEFT JOIN conversations c ON c.id = m.conversation_id`)
		where = append(where, "message_fts MATCH ?")
		args = append(args, ToFTS5(terms))
	} else {
		b.WriteString(`SELECT m.id, m.key, m.sent_at, m.from_name, coalesce(m.sender_url, ''),
			coalesce(m.to_names, ''), coalesce(m.folder, ''), m.subject,
			substr(m.body_text, 1, 200), 0.0,
			coalesce(c.conversation_id, ''), coalesce(c.title, '')
		FROM messages m
		LEFT JOIN conversations c ON c.id = m.conversation_id`)
	}
	if q.Folder != "" {
		where = append(where, "m.folder = ? COLLATE NOCASE")
		args = append(args, q.Folder)
	}
	if q.From != "" {
		where = append(where, "(m.from_name LIKE ? OR m.sender_url LIKE ?)")
		like := "%" + q.From + "%"
		args = append(args, like, like)
	}
	if !q.Since.IsZero() {
		where = append(where, "m.sent_at >= ?")
		args = append(args, q.Since.UTC())
	}
	if !q.Until.IsZero() {
		where = append(where, "m.sent_at < ?")
		args = append(args, q.Until.UTC())
	}
	if len(where) > 0 {
		b.WriteString(" WHERE " + strings.Join(where, " AND "))
	}
	if len(terms) > 0 {
		b.WriteString(" ORDER BY bm25(message_fts)")
	} else {
		b.WriteString(" ORDER BY m.sent_at DESC")
	}
	b.WriteString(" LIMIT ?")
	args = append(args, q.EffectiveLimit())
	return query(ctx, s.db, b.String(), args)
}

func query(ctx context.Context, db *sql.DB, stmt string, args []any) (out []search.Result, err error) {
	rows, err := db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	for rows.Next() {
		var (
			r      search.Result
			sentAt sql.NullTime
		)
		if err := rows.Scan(&r.MessageID, &r.Key, &sentAt, &r.From, &r.SenderURL, &r.To,
			&r.Folder, &r.Subject, &r.Snippet, &r.Rank, &r.ConversationID, &r.ConversationTitle); err != nil {
			return nil, err
		}
		if sentAt.Valid {
			t := sentAt.Time.UTC()
			r.SentAt = &t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ToFTS5 renders parsed terms as an FTS5 MATCH expression. Every word and
// phrase is quoted so user input cannot inject FTS5 syntax. Excluded terms
// are applied with FTS5's binary NOT to the group they appear in.
func ToFTS5(terms []search.Term) string {
	var pos strings.Builder
	var neg []string
	for _, t := range terms {
		var s string
		if t.Group != nil {
			s = ToFTS5(t.Group)
		} else {
			s = `"` + strings.ReplaceAll(t.Text, `"`, `""`) + `"`
			if t.Prefix && !t.Phrase {
				s += "*"
			}
		}
		if t.Negate {
			neg = append(neg, s)
			continue
		}
		if pos.Len() > 0 {
			if t.Or {
				pos.WriteString(" OR ")
			} else {
				pos.WriteString(" AND ")
			}
		}
		pos.WriteString(s)
	}
	expr := "(" + pos.String() + ")"
	for _, n := range neg {
		expr = "(" + expr + " NOT " + n + ")"
	}
	return expr
}
