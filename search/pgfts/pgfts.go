// Package pgfts implements message search with PostgreSQL full-text search.
//
// It uses a GIN expression index over the messages table, so no separate
// index table or sync step is needed. Queries are built from tsquery values
// combined with &&, || and !!, with every user term passed as a parameter.
package pgfts

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/grokify/linkedin-archive/search"
)

const (
	config = "english"

	// document is the searchable text of a message aliased as m. It must
	// match indexDocument (minus the alias) for PostgreSQL to use the index.
	document = `to_tsvector('english', coalesce(m.subject, '') || ' ' || coalesce(m.body_text, '') || ' ' || coalesce(m.from_name, ''))`

	indexDocument = `to_tsvector('english', coalesce(subject, '') || ' ' || coalesce(body_text, '') || ' ' || coalesce(from_name, ''))`

	createIndexSQL = `CREATE INDEX IF NOT EXISTS messages_fts_idx ON messages USING GIN (` + indexDocument + `)`
)

// Searcher searches messages in a PostgreSQL database.
type Searcher struct {
	db *sql.DB
}

// New returns a Searcher, creating the full-text index if needed.
func New(ctx context.Context, db *sql.DB) (*Searcher, error) {
	if _, err := db.ExecContext(ctx, createIndexSQL); err != nil {
		return nil, fmt.Errorf("create fts index: %w", err)
	}
	return &Searcher{db: db}, nil
}

// Search implements search.Searcher.
func (s *Searcher) Search(ctx context.Context, q search.Query) (out []search.Result, err error) {
	terms, err := search.Parse(q.Text)
	if err != nil {
		return nil, err
	}
	var (
		where []string
		args  []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	snippet, rank, order := `left(m.body_text, 200)`, `0.0`, `m.sent_at DESC NULLS LAST`
	if len(terms) > 0 {
		tq := ToTSQuery(terms, arg)
		where = append(where, document+" @@ "+tq)
		snippet = fmt.Sprintf(`ts_headline('%s', m.body_text, %s, 'StartSel=[,StopSel=],MaxWords=35,MinWords=15')`, config, tq)
		rank = fmt.Sprintf(`ts_rank(%s, %s)`, document, tq)
		order = "rank DESC"
	}
	if q.Folder != "" {
		where = append(where, "upper(m.folder) = upper("+arg(q.Folder)+")")
	}
	if q.From != "" {
		p := arg("%" + q.From + "%")
		where = append(where, "(m.from_name ILIKE "+p+" OR m.sender_url ILIKE "+p+")")
	}
	if !q.Since.IsZero() {
		where = append(where, "m.sent_at >= "+arg(q.Since.UTC()))
	}
	if !q.Until.IsZero() {
		where = append(where, "m.sent_at < "+arg(q.Until.UTC()))
	}
	stmt := `SELECT m.id, m.key, m.sent_at, m.from_name, coalesce(m.sender_url, ''),
		coalesce(m.to_names, ''), coalesce(m.folder, ''), m.subject,
		` + snippet + `, ` + rank + ` AS rank,
		coalesce(c.conversation_id, ''), coalesce(c.title, '')
	FROM messages m
	LEFT JOIN conversations c ON c.id = m.conversation_id`
	if len(where) > 0 {
		stmt += " WHERE " + strings.Join(where, " AND ")
	}
	// Every fragment above is a constant or a generated $n placeholder; all
	// user input is bound through arg.
	stmt += " ORDER BY " + order + " LIMIT " + arg(q.EffectiveLimit()) //nolint:gosec // G202: Dynamic clauses are constant fragments; all user values are bound parameters

	rows, err := s.db.QueryContext(ctx, stmt, args...)
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

var nonWord = regexp.MustCompile(`[^\pL\pN_]+`)

// ToTSQuery renders parsed terms as a tsquery SQL expression. Each term's
// text is bound through arg, which returns its placeholder. Grouping is made
// explicit because PostgreSQL gives && and || equal precedence: AND binds
// tighter than OR, and excluded terms apply to their whole group, matching
// the SQLite implementation.
func ToTSQuery(terms []search.Term, arg func(any) string) string {
	var (
		clauses [][]string // OR-separated clauses of ANDed terms
		neg     []string
	)
	for _, t := range terms {
		s := tsTerm(t, arg)
		if t.Negate {
			neg = append(neg, "!!"+s)
			continue
		}
		if t.Or || len(clauses) == 0 {
			clauses = append(clauses, nil)
		}
		clauses[len(clauses)-1] = append(clauses[len(clauses)-1], s)
	}
	ors := make([]string, len(clauses))
	for i, c := range clauses {
		ors[i] = "(" + strings.Join(c, " && ") + ")"
	}
	expr := "(" + strings.Join(ors, " || ") + ")"
	for _, n := range neg {
		expr = "(" + expr + " && " + n + ")"
	}
	return expr
}

func tsTerm(t search.Term, arg func(any) string) string {
	switch {
	case t.Group != nil:
		return ToTSQuery(t.Group, arg)
	case t.Phrase:
		return fmt.Sprintf("phraseto_tsquery('%s', %s)", config, arg(t.Text))
	case t.Prefix:
		if w := nonWord.ReplaceAllString(t.Text, ""); w != "" {
			return fmt.Sprintf("to_tsquery('%s', %s)", config, arg(w+":*"))
		}
	}
	return fmt.Sprintf("plainto_tsquery('%s', %s)", config, arg(t.Text))
}
