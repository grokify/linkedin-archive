package importer_test

import (
	"context"
	"testing"

	"github.com/grokify/linkedin-archive/archive"
	"github.com/grokify/linkedin-archive/ent/message"
	"github.com/grokify/linkedin-archive/importer"
	"github.com/grokify/linkedin-archive/internal/testfixture"
	"github.com/grokify/linkedin-archive/search"
	"github.com/grokify/linkedin-archive/search/engine"
	"github.com/grokify/linkedin-archive/store"
)

func TestImportAndSearchSQLite(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, "sqlite::memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	RunImportAndSearch(t, s)
}

// RunImportAndSearch exercises the import/search lifecycle against s.
func RunImportAndSearch(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()
	open := func(fsys archiveFS, name string) *archive.Archive {
		a, err := archive.New(fsys, name)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}

	// Create the search index before importing, so triggers keep it in sync.
	srch, err := engine.New(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	res, err := importer.Import(ctx, s.Client, open(testfixture.Basic(), "basic"))
	if err != nil {
		t.Fatal(err)
	}
	if res.AlreadyImported || res.Kind != archive.KindBasic {
		t.Fatalf("result = %+v", res)
	}
	want := map[string][3]int{ // seen, inserted, skipped
		"messages":       {4, 4, 0},
		"conversations":  {3, 3, 0},
		"connections":    {1, 1, 1},
		"invitations":    {1, 1, 0},
		"raw:Skills.csv": {2, 2, 0},
	}
	for k, w := range want {
		got := res.Stats[k]
		if got.Seen != w[0] || got.Inserted != w[1] || got.Skipped != w[2] {
			t.Errorf("stats[%s] = %+v, want %v", k, got, w)
		}
	}

	again, err := importer.Import(ctx, s.Client, open(testfixture.Basic(), "basic-again"))
	if err != nil {
		t.Fatal(err)
	}
	if !again.AlreadyImported || again.ImportID != res.ImportID {
		t.Fatalf("re-import = %+v", again)
	}

	newer, err := importer.Import(ctx, s.Client, open(testfixture.Newer(), "newer"))
	if err != nil {
		t.Fatal(err)
	}
	if newer.Kind != archive.KindComplete {
		t.Errorf("kind = %s", newer.Kind)
	}
	if got := newer.Stats["messages"]; got.Seen != 5 || got.Inserted != 1 {
		t.Errorf("newer messages = %+v", got)
	}
	if got := newer.Stats["saved_items"]; got.Inserted != 1 {
		t.Errorf("saved items = %+v", got)
	}
	n, err := s.Client.Message.Query().Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("messages = %d, want 5", n)
	}
	moved, err := s.Client.Message.Query().Where(message.FromName("Pat Owner")).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Folder != "ARCHIVE" || moved.FirstSeenImportID != res.ImportID || moved.LastSeenImportID != newer.ImportID {
		t.Errorf("moved message = folder %s first %d last %d", moved.Folder, moved.FirstSeenImportID, moved.LastSeenImportID)
	}
	sender, err := moved.QuerySender().Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sender.ProfileURL != "https://www.linkedin.com/in/pat-owner" {
		t.Errorf("sender = %+v", sender)
	}

	searches := []struct {
		q    search.Query
		want []string // senders, in any order
	}{
		{search.Query{Text: `"example corp" (shares OR options OR tender)`}, []string{"Equity Buyer", "Liquidity Fund", "Tender Desk"}},
		{search.Query{Text: "shares -tender"}, []string{"Equity Buyer"}},
		{search.Query{Text: "example", Folder: "archive"}, []string{"Liquidity Fund"}},
		{search.Query{Text: "example", From: "tender-desk"}, []string{"Tender Desk"}},
		{search.Query{Text: "hiring OR shares", Until: mustDate(t, "2026-01-01")}, []string{"Recruiter Person"}},
		{search.Query{From: "Pat"}, []string{"Pat Owner"}},
	}
	for _, c := range searches {
		results, err := srch.Search(ctx, c.q)
		if err != nil {
			t.Fatalf("search %+v: %v", c.q, err)
		}
		got := map[string]bool{}
		for _, r := range results {
			got[r.From] = true
			if r.Snippet == "" || r.ConversationID == "" {
				t.Errorf("search %+v: incomplete result %+v", c.q, r)
			}
		}
		if len(got) != len(c.want) {
			t.Errorf("search %+v: got %v, want %v", c.q, got, c.want)
			continue
		}
		for _, w := range c.want {
			if !got[w] {
				t.Errorf("search %+v: missing %s in %v", c.q, w, got)
			}
		}
	}
}
