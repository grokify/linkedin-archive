package archive

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Message is one row of messages.csv or a *_messages.csv file.
type Message struct {
	Key            string // stable identity across exports
	ConversationID string
	Title          string
	From           string
	SenderURL      string
	To             string // verbatim; names may contain commas
	RecipientURLs  []string
	SentAt         time.Time
	Subject        string
	Content        string // raw, may contain HTML
	BodyText       string // HTML stripped, whitespace collapsed
	Folder         string
	Attachments    string
	SourceFile     string
}

// Connection is one row of Connections.csv.
type Connection struct {
	FirstName   string
	LastName    string
	URL         string
	Email       string
	Company     string
	Position    string
	ConnectedOn time.Time
}

// Name returns the connection's full name.
func (c Connection) Name() string {
	return strings.TrimSpace(c.FirstName + " " + c.LastName)
}

// Invitation is one row of Invitations.csv.
type Invitation struct {
	Key        string
	From       string
	To         string
	SentAt     time.Time
	Message    string
	Direction  string
	InviterURL string
	InviteeURL string
}

// SavedItem is one row of a saved items/posts file (Complete export).
type SavedItem struct {
	Key        string
	URL        string
	SavedAt    time.Time
	Title      string
	Author     string
	Content    string
	Extra      map[string]string // columns not mapped above
	SourceFile string
}

var messageHeader = []string{"CONVERSATION ID", "FROM", "DATE", "CONTENT"}

// MessageFiles returns messages.csv and any *_messages.csv files.
func (a *Archive) MessageFiles() []string {
	return a.Match(func(b string) bool {
		return b == "messages.csv" || strings.HasSuffix(b, "_messages.csv")
	})
}

// Messages parses all message files. Keys are unique within the export:
// identical rows receive an occurrence suffix.
func (a *Archive) Messages() ([]Message, error) {
	var out []Message
	seen := map[string]int{}
	for _, f := range a.MessageFiles() {
		t, err := a.ReadCSV(f, messageHeader...)
		if err != nil {
			return nil, err
		}
		for _, r := range t.Rows {
			m := Message{
				ConversationID: r.Get("CONVERSATION ID"),
				Title:          r.Get("CONVERSATION TITLE"),
				From:           r.Get("FROM"),
				SenderURL:      NormalizeProfileURL(r.Get("SENDER PROFILE URL")),
				To:             r.Get("TO"),
				RecipientURLs:  splitURLs(r.Get("RECIPIENT PROFILE URLS")),
				Subject:        r.Get("SUBJECT"),
				Content:        r["CONTENT"],
				Folder:         r.Get("FOLDER"),
				Attachments:    r.Get("ATTACHMENTS"),
				SourceFile:     path.Base(f),
			}
			m.SentAt, _ = ParseTime(r.Get("DATE")) // zero time when absent or unparseable
			m.BodyText = HTMLToText(m.Content)
			base := HashKey(m.ConversationID, r.Get("DATE"), m.SenderURL, m.From, m.Subject, m.Content)
			seen[base]++
			m.Key = base
			if n := seen[base]; n > 1 {
				m.Key = base + "-" + strconv.Itoa(n)
			}
			out = append(out, m)
		}
	}
	return out, nil
}

// Connections parses Connections.csv. It returns nil if the file is absent.
func (a *Archive) Connections() ([]Connection, error) {
	f := a.Find("Connections.csv")
	if f == "" {
		return nil, nil
	}
	t, err := a.ReadCSV(f, "First Name", "Last Name")
	if err != nil {
		return nil, err
	}
	out := make([]Connection, 0, len(t.Rows))
	for _, r := range t.Rows {
		c := Connection{
			FirstName: r.Get("First Name"),
			LastName:  r.Get("Last Name"),
			URL:       NormalizeProfileURL(r.Get("URL")),
			Email:     r.Get("Email Address"),
			Company:   r.Get("Company"),
			Position:  r.Get("Position"),
		}
		c.ConnectedOn, _ = ParseTime(r.Get("Connected On")) // zero time when absent
		out = append(out, c)
	}
	return out, nil
}

// Invitations parses Invitations.csv. It returns nil if the file is absent.
func (a *Archive) Invitations() ([]Invitation, error) {
	f := a.Find("Invitations.csv")
	if f == "" {
		return nil, nil
	}
	t, err := a.ReadCSV(f, "From", "To")
	if err != nil {
		return nil, err
	}
	out := make([]Invitation, 0, len(t.Rows))
	for _, r := range t.Rows {
		inv := Invitation{
			From:       r.Get("From"),
			To:         r.Get("To"),
			Message:    r.Get("Message"),
			Direction:  r.Get("Direction"),
			InviterURL: NormalizeProfileURL(r.Get("inviterProfileUrl")),
			InviteeURL: NormalizeProfileURL(r.Get("inviteeProfileUrl")),
		}
		inv.SentAt, _ = ParseTime(r.Get("Sent At")) // zero time when absent
		inv.Key = HashKey(inv.From, inv.To, r.Get("Sent At"), inv.Direction, inv.InviterURL, inv.InviteeURL, inv.Message)
		out = append(out, inv)
	}
	return out, nil
}

// SavedItemFiles returns files that look like saved items or saved posts.
func (a *Archive) SavedItemFiles() []string {
	re := regexp.MustCompile(`^saved[ _-]?(items?|posts?|articles?)\.csv$`)
	return a.Match(re.MatchString)
}

var (
	savedURLCols     = []string{"Saved Item", "SavedItem", "Url", "URL", "Link", "Post Url", "Post URL", "Item Url", "Content Url"}
	savedDateCols    = []string{"Saved At", "Saved Date", "Saved On", "Date Saved", "Created Time", "Creation Date", "Date"}
	savedTitleCols   = []string{"Title", "Headline"}
	savedAuthorCols  = []string{"Author", "Author Name", "Actor"}
	savedContentCols = []string{"Content", "Commentary", "Text", "Description"}
)

// SavedItems parses saved items/posts files. Columns are mapped by header
// name; unmapped columns are kept in Extra.
func (a *Archive) SavedItems() ([]SavedItem, error) {
	var out []SavedItem
	for _, f := range a.SavedItemFiles() {
		t, err := a.ReadCSV(f)
		if err != nil {
			return nil, err
		}
		for _, r := range t.Rows {
			it := SavedItem{SourceFile: path.Base(f), Extra: map[string]string{}}
			used := map[string]bool{}
			pick := func(cols []string) string {
				for _, c := range cols {
					for _, h := range t.Header {
						if strings.EqualFold(h, c) && !used[h] {
							used[h] = true
							return strings.TrimSpace(r[h])
						}
					}
				}
				return ""
			}
			it.URL = pick(savedURLCols)
			dateStr := pick(savedDateCols)
			it.SavedAt, _ = ParseTime(dateStr) // zero time when absent
			it.Title = pick(savedTitleCols)
			it.Author = pick(savedAuthorCols)
			it.Content = pick(savedContentCols)
			if it.URL == "" {
				// Fall back to the first value that looks like a URL.
				for _, h := range t.Header {
					if v := strings.TrimSpace(r[h]); !used[h] && strings.HasPrefix(v, "http") {
						it.URL, used[h] = v, true
						break
					}
				}
			}
			for _, h := range t.Header {
				if !used[h] && strings.TrimSpace(r[h]) != "" {
					it.Extra[h] = r[h]
				}
			}
			it.Key = HashKey(it.URL, dateStr, it.Title)
			out = append(out, it)
		}
	}
	return out, nil
}

// NormalizeProfileURL canonicalizes a LinkedIn profile URL: https scheme,
// www host, no trailing slash, query or fragment.
func NormalizeProfileURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	u = strings.TrimRight(u, "/")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "https://")
	if strings.HasPrefix(u, "linkedin.com/") {
		u = "www." + u
	}
	return "https://" + u
}

func splitURLs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if u := NormalizeProfileURL(p); u != "" {
			out = append(out, u)
		}
	}
	return out
}

var (
	reBlockTag = regexp.MustCompile(`(?i)<\s*(br|/p|/div|/li|/h[1-6])\s*/?>`)
	reTag      = regexp.MustCompile(`<[^>]*>`)
	reSpace    = regexp.MustCompile(`\s+`)
)

// HTMLToText strips HTML tags, unescapes entities and collapses whitespace.
// Plain text passes through with whitespace collapsed.
func HTMLToText(s string) string {
	if strings.Contains(s, "<") {
		s = reBlockTag.ReplaceAllString(s, " ")
		s = reTag.ReplaceAllString(s, " ")
	}
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, " ", " ")
	return strings.TrimSpace(reSpace.ReplaceAllString(s, " "))
}

// HashKey returns a short, stable hex SHA-256 of the given parts.
func HashKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}
