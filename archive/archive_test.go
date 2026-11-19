package archive

import (
	"testing"
	"testing/fstest"
	"time"
)

const testMessages = `"CONVERSATION ID","CONVERSATION TITLE","FROM","SENDER PROFILE URL","TO","RECIPIENT PROFILE URLS","DATE","SUBJECT","CONTENT","FOLDER","ATTACHMENTS"
"c1","","Alice Example","https://www.linkedin.com/in/alice-example/","Pat Owner","https://www.linkedin.com/in/pat-owner","2026-01-02 03:04:05 UTC","Hello","<p>Hi Pat,</p><p>Interested in your &amp; equity?</p>","INBOX",""
"c1","","Pat Owner","https://www.linkedin.com/in/pat-owner","Alice Example","https://www.linkedin.com/in/alice-example","2026-01-02 04:00:00 UTC","","No thanks","INBOX",""
"c1","","Pat Owner","https://www.linkedin.com/in/pat-owner","Alice Example","https://www.linkedin.com/in/alice-example","2026-01-02 04:00:00 UTC","","No thanks","INBOX",""
"c2","","Sponsor Co","","Pat Owner, MBA,Bob Example","https://www.linkedin.com/in/pat-owner,https://www.linkedin.com/in/bob-example","2026-02-01 10:00:00 UTC","","Buy now","SPAM",""
`

const testConnections = `Notes:
"Some note about emails."

First Name,Last Name,URL,Email Address,Company,Position,Connected On
Alice,Example,https://www.linkedin.com/in/alice-example,,Acme,CEO,31 Oct 2026
`

const testInvitations = `From,To,Sent At,Message,Direction,inviterProfileUrl,inviteeProfileUrl
Bob Example,Pat Owner,"10/30/26, 5:04 PM",,INCOMING,https://www.linkedin.com/in/bob-example,https://www.linkedin.com/in/pat-owner
`

const testSaved = `Saved Item,Saved At,Note
https://www.linkedin.com/feed/update/urn:li:activity:1,2026-03-04 05:06:07 UTC,read later
`

func testArchive(t *testing.T) *Archive {
	t.Helper()
	fsys := fstest.MapFS{
		"messages.csv":                {Data: []byte(testMessages)},
		"Connections.csv":             {Data: []byte("\xef\xbb\xbf" + testConnections)},
		"Invitations.csv":             {Data: []byte(testInvitations)},
		"Saved_Items.csv":             {Data: []byte(testSaved)},
		"Jobs/Saved Jobs.csv":         {Data: []byte("Saved Date,Job Url\n\"12/12/22, 3:48 PM\",https://example.com\n")},
		"__MACOSX/._messages.csv":     {Data: []byte("junk")},
		"learning_coach_messages.csv": {Data: []byte(`CONVERSATION ID,CONVERSATION TITLE,FROM,SENDER PROFILE URL,TO,RECIPIENT PROFILE URLS,DATE,SUBJECT,CONTENT,FOLDER` + "\n")},
	}
	a, err := New(fsys, "test")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestFilesAndKind(t *testing.T) {
	a := testArchive(t)
	if got := len(a.Files()); got != 6 {
		t.Fatalf("files = %d, want 6: %v", got, a.Files())
	}
	if a.Kind() != KindBasic {
		t.Errorf("kind = %s", a.Kind())
	}
	if len(a.SHA256()) != 64 {
		t.Errorf("sha256 = %q", a.SHA256())
	}
	if got := len(a.MessageFiles()); got != 2 {
		t.Errorf("message files = %d", got)
	}
}

func TestMessages(t *testing.T) {
	ms, err := testArchive(t).Messages()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 4 {
		t.Fatalf("messages = %d, want 4", len(ms))
	}
	m := ms[0]
	if m.BodyText != "Hi Pat, Interested in your & equity?" {
		t.Errorf("body = %q", m.BodyText)
	}
	if m.SenderURL != "https://www.linkedin.com/in/alice-example" {
		t.Errorf("sender url = %q", m.SenderURL)
	}
	if !m.SentAt.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("sent at = %v", m.SentAt)
	}
	if ms[1].Key == ms[2].Key {
		t.Error("duplicate rows must get distinct keys")
	}
	if len(ms[3].RecipientURLs) != 2 {
		t.Errorf("recipients = %v", ms[3].RecipientURLs)
	}
	again, err := testArchive(t).Messages()
	if err != nil {
		t.Fatal(err)
	}
	if again[2].Key != ms[2].Key {
		t.Error("keys must be stable across reads")
	}
}

func TestConnectionsInvitationsSaved(t *testing.T) {
	a := testArchive(t)
	cs, err := a.Connections()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 || cs[0].Name() != "Alice Example" || cs[0].ConnectedOn.Year() != 2026 {
		t.Fatalf("connections = %+v", cs)
	}
	invs, err := a.Invitations()
	if err != nil {
		t.Fatal(err)
	}
	if len(invs) != 1 || invs[0].SentAt.Hour() != 17 {
		t.Fatalf("invitations = %+v", invs)
	}
	saved, err := a.SavedItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 {
		t.Fatalf("saved = %+v", saved)
	}
	s := saved[0]
	if s.URL == "" || s.SavedAt.IsZero() || s.Extra["Note"] != "read later" {
		t.Errorf("saved item = %+v", s)
	}
}

func TestParseTime(t *testing.T) {
	cases := map[string]time.Time{
		"2026-11-17 20:18:55 UTC":      time.Date(2026, 11, 17, 20, 18, 55, 0, time.UTC),
		"2025/10/08 11:50:14 UTC":      time.Date(2025, 10, 8, 11, 50, 14, 0, time.UTC),
		"2026-04-26 14:07 UTC":         time.Date(2026, 4, 26, 14, 7, 0, 0, time.UTC),
		"31 Oct 2026":                  time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		"10/30/26, 5:04 PM":            time.Date(2026, 10, 30, 17, 4, 0, 0, time.UTC),
		"Wed Nov 18 20:32:07 GMT 2026": time.Date(2026, 11, 18, 20, 32, 7, 0, time.UTC),
		"Jul 2023":                     time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC),
		"You uploaded a feed photo on June 24, 2026 at 10:33 PM (GMT)": time.Date(2026, 6, 24, 22, 33, 0, 0, time.UTC),
	}
	for in, want := range cases {
		got, ok := ParseTime(in)
		if !ok || !got.Equal(want) {
			t.Errorf("ParseTime(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "N/A", "not a date"} {
		if _, ok := ParseTime(in); ok {
			t.Errorf("ParseTime(%q) ok = true", in)
		}
	}
}

func TestParseCSVDuplicateHeaders(t *testing.T) {
	tb, err := ParseCSV([]byte("A,A,,B\n1,2,3,4\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A", "A_2", "column_3", "B"}
	for i, h := range want {
		if tb.Header[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, tb.Header[i], h)
		}
	}
	if tb.Rows[0]["A_2"] != "2" {
		t.Errorf("row = %v", tb.Rows[0])
	}
}

func TestNormalizeProfileURL(t *testing.T) {
	cases := map[string]string{
		"https://www.linkedin.com/in/x/":   "https://www.linkedin.com/in/x",
		"http://linkedin.com/in/x?trk=abc": "https://www.linkedin.com/in/x",
		" ":                                "",
	}
	for in, want := range cases {
		if got := NormalizeProfileURL(in); got != want {
			t.Errorf("NormalizeProfileURL(%q) = %q, want %q", in, got, want)
		}
	}
}
