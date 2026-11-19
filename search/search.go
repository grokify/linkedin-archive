// Package search defines engine-neutral message search types and the
// portable query syntax. Engine implementations live in subpackages
// (sqlitefts, pgfts); use the engine package to pick one for a store.
package search

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
)

// DefaultLimit is the result limit used when Query.Limit is zero.
const DefaultLimit = 50

// Query is a message search request.
type Query struct {
	Text   string    // query in the portable syntax; empty lists by filters only
	From   string    // substring of the sender name or profile URL
	Folder string    // exact folder, e.g. INBOX, ARCHIVE, SPAM; empty = all
	Since  time.Time // inclusive; zero = unbounded
	Until  time.Time // exclusive; zero = unbounded
	Limit  int
}

// EffectiveLimit returns Limit, or DefaultLimit when Limit is not positive.
func (q Query) EffectiveLimit() int {
	if q.Limit <= 0 {
		return DefaultLimit
	}
	return q.Limit
}

// Result is one matching message.
type Result struct {
	MessageID         int        `json:"messageId"`
	Key               string     `json:"key"`
	ConversationID    string     `json:"conversationId,omitempty"`
	ConversationTitle string     `json:"conversationTitle,omitempty"`
	SentAt            *time.Time `json:"sentAt,omitempty"`
	From              string     `json:"from"`
	SenderURL         string     `json:"senderUrl,omitempty"`
	To                string     `json:"to,omitempty"`
	Folder            string     `json:"folder,omitempty"`
	Subject           string     `json:"subject,omitempty"`
	Snippet           string     `json:"snippet"`
	Rank              float64    `json:"rank"`
}

// Searcher searches messages.
type Searcher interface {
	Search(ctx context.Context, q Query) ([]Result, error)
}

// Term is one element of a parsed query: a word, a phrase, or a
// parenthesized group of terms.
type Term struct {
	Text   string
	Phrase bool   // quoted phrase
	Prefix bool   // trailing *
	Negate bool   // leading -
	Or     bool   // joined to the previous term with OR instead of AND
	Group  []Term // non-nil for a parenthesized group
}

// Errors returned by Parse.
var (
	ErrOnlyNegated    = errors.New("query and each group need at least one term that is not excluded")
	ErrUnbalanced     = errors.New("unbalanced parentheses in query")
	ErrNestingTooDeep = errors.New("query nesting too deep")
)

const maxDepth = 8

// Parse parses the portable syntax: bare words (ANDed), "quoted phrases",
// OR between terms, -excluded terms, prefix* matches and (groups).
// AND binds tighter than OR, so "a b OR c" means "(a AND b) OR c".
func Parse(s string) ([]Term, error) {
	p := &parser{rs: []rune(s)}
	terms, err := p.parse(0)
	if err != nil {
		return nil, err
	}
	if p.i < len(p.rs) {
		return nil, ErrUnbalanced // stray ")"
	}
	return terms, nil
}

type parser struct {
	rs []rune
	i  int
}

func (p *parser) parse(depth int) ([]Term, error) {
	if depth > maxDepth {
		return nil, ErrNestingTooDeep
	}
	var terms []Term
	pendingOr := false
	rs := p.rs
	for p.i < len(rs) {
		c := rs[p.i]
		if unicode.IsSpace(c) {
			p.i++
			continue
		}
		if c == ')' {
			if depth == 0 {
				return nil, ErrUnbalanced
			}
			break
		}
		t := Term{}
		if c == '-' && p.i+1 < len(rs) && !unicode.IsSpace(rs[p.i+1]) {
			t.Negate = true
			p.i++
		}
		switch rs[p.i] {
		case '(':
			p.i++
			g, err := p.parse(depth + 1)
			if err != nil {
				return nil, err
			}
			if p.i >= len(rs) || rs[p.i] != ')' {
				return nil, ErrUnbalanced
			}
			p.i++
			if len(g) == 0 {
				continue
			}
			t.Group = g
		case '"':
			j := p.i + 1
			for j < len(rs) && rs[j] != '"' {
				j++
			}
			t.Text = strings.Join(strings.Fields(string(rs[p.i+1:min(j, len(rs))])), " ")
			t.Phrase = true
			p.i = j + 1
		default:
			j := p.i
			for j < len(rs) && !unicode.IsSpace(rs[j]) && !strings.ContainsRune(`"()`, rs[j]) {
				j++
			}
			t.Text = string(rs[p.i:j])
			p.i = j
			if !t.Negate && t.Text == "OR" {
				pendingOr = len(terms) > 0
				continue
			}
			if strings.HasSuffix(t.Text, "*") {
				t.Text = strings.TrimRight(t.Text, "*")
				t.Prefix = true
			}
		}
		if t.Text == "" && t.Group == nil {
			continue
		}
		if pendingOr && !t.Negate {
			t.Or = true
		}
		pendingOr = false
		terms = append(terms, t)
	}
	if len(terms) > 0 && !hasPositive(terms) {
		return nil, ErrOnlyNegated
	}
	return terms, nil
}

func hasPositive(terms []Term) bool {
	for _, t := range terms {
		if !t.Negate {
			return true
		}
	}
	return false
}
