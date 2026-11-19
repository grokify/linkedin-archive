// Package importer loads LinkedIn export archives into the database.
// Imports are idempotent: an archive already imported is skipped, and records
// already present are matched by natural key and only have their
// last-seen import updated.
package importer

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/grokify/linkedin-archive/archive"
	"github.com/grokify/linkedin-archive/ent"
	"github.com/grokify/linkedin-archive/ent/connection"
	"github.com/grokify/linkedin-archive/ent/conversation"
	"github.com/grokify/linkedin-archive/ent/importrun"
	"github.com/grokify/linkedin-archive/ent/invitation"
	"github.com/grokify/linkedin-archive/ent/message"
	"github.com/grokify/linkedin-archive/ent/person"
	"github.com/grokify/linkedin-archive/ent/rawrecord"
	"github.com/grokify/linkedin-archive/ent/saveditem"
	"github.com/grokify/linkedin-archive/ent/schema"
)

// Result summarizes an import.
type Result struct {
	ImportID        int
	AlreadyImported bool
	Kind            archive.Kind
	Stats           map[string]schema.FileStats
}

// Import loads a into the database behind client.
func Import(ctx context.Context, client *ent.Client, a *archive.Archive) (*Result, error) {
	existing, err := client.ImportRun.Query().Where(importrun.Sha256(a.SHA256())).Only(ctx)
	switch {
	case err == nil:
		return &Result{ImportID: existing.ID, AlreadyImported: true, Kind: archive.Kind(existing.ExportKind), Stats: existing.Stats}, nil
	case !ent.IsNotFound(err):
		return nil, fmt.Errorf("look up import: %w", err)
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	res, err := importTx(ctx, tx, a)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = errors.Join(err, fmt.Errorf("rollback: %w", rerr))
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return res, nil
}

type run struct {
	ctx     context.Context
	tx      *ent.Tx
	id      int
	people  map[string]int
	convs   map[string]int
	stats   map[string]schema.FileStats
	handled map[string]bool
}

func importTx(ctx context.Context, tx *ent.Tx, a *archive.Archive) (*Result, error) {
	ir, err := tx.ImportRun.Create().
		SetSha256(a.SHA256()).
		SetSourceName(a.Name()).
		SetExportKind(string(a.Kind())).
		SetFiles(a.Files()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create import: %w", err)
	}
	r := &run{
		ctx: ctx, tx: tx, id: ir.ID,
		people:  map[string]int{},
		convs:   map[string]int{},
		stats:   map[string]schema.FileStats{},
		handled: map[string]bool{},
	}
	steps := []func(*archive.Archive) error{
		r.connections, r.messages, r.invitations, r.savedItems, r.rawRecords,
	}
	for _, step := range steps {
		if err := step(a); err != nil {
			return nil, err
		}
	}
	if err := tx.ImportRun.UpdateOneID(ir.ID).SetStats(r.stats).Exec(ctx); err != nil {
		return nil, fmt.Errorf("save stats: %w", err)
	}
	return &Result{ImportID: ir.ID, Kind: a.Kind(), Stats: r.stats}, nil
}

func (r *run) count(entity string, inserted bool) {
	s := r.stats[entity]
	s.Seen++
	if inserted {
		s.Inserted++
	}
	r.stats[entity] = s
}

// personKey returns the identity key for a person.
func personKey(url, name string) string {
	if url != "" {
		return url
	}
	name = strings.ToLower(strings.Join(strings.Fields(name), " "))
	if name == "" {
		return ""
	}
	return "name:" + name
}

// person returns the ID of the person with the given URL/name, creating it
// if needed. It returns 0 when neither is known.
func (r *run) person(url, name string) (int, error) {
	key := personKey(url, name)
	if key == "" {
		return 0, nil
	}
	if id, ok := r.people[key]; ok {
		return id, nil
	}
	p, err := r.tx.Person.Query().Where(person.Key(key)).Only(r.ctx)
	switch {
	case err == nil:
		if p.Name == "" && name != "" {
			if err := r.tx.Person.UpdateOneID(p.ID).SetName(name).Exec(r.ctx); err != nil {
				return 0, err
			}
		}
	case ent.IsNotFound(err):
		c := r.tx.Person.Create().SetKey(key).SetName(name)
		if url != "" {
			c.SetProfileURL(url)
		}
		if p, err = c.Save(r.ctx); err != nil {
			return 0, fmt.Errorf("create person %s: %w", key, err)
		}
	default:
		return 0, err
	}
	r.people[key] = p.ID
	return p.ID, nil
}

func (r *run) connections(a *archive.Archive) error {
	cs, err := a.Connections()
	if err != nil {
		return err
	}
	if f := a.Find("Connections.csv"); f != "" {
		r.handled[f] = true
	}
	for _, c := range cs {
		pid, err := r.person(c.URL, c.Name())
		if err != nil {
			return err
		}
		if pid == 0 {
			// Deactivated accounts appear with only a connection date.
			s := r.stats["connections"]
			s.Skipped++
			r.stats["connections"] = s
			continue
		}
		existing, err := r.tx.Connection.Query().Where(connection.PersonID(pid)).Only(r.ctx)
		switch {
		case err == nil:
			err = r.tx.Connection.UpdateOne(existing).
				SetLastSeenImportID(r.id).
				SetCompany(c.Company).
				SetPosition(c.Position).
				SetEmail(c.Email).
				Exec(r.ctx)
			r.count("connections", false)
		case ent.IsNotFound(err):
			err = r.tx.Connection.Create().
				SetPersonID(pid).
				SetFirstName(c.FirstName).
				SetLastName(c.LastName).
				SetProfileURL(c.URL).
				SetEmail(c.Email).
				SetCompany(c.Company).
				SetPosition(c.Position).
				SetNillableConnectedOn(timePtr(c.ConnectedOn)).
				SetFirstSeenImportID(r.id).
				SetLastSeenImportID(r.id).
				Exec(r.ctx)
			r.count("connections", true)
		}
		if err != nil {
			return fmt.Errorf("connection %s: %w", c.Name(), err)
		}
	}
	return nil
}

func (r *run) conversation(m archive.Message) (int, error) {
	if m.ConversationID == "" {
		return 0, nil
	}
	if id, ok := r.convs[m.ConversationID]; ok {
		return id, nil
	}
	c, err := r.tx.Conversation.Query().Where(conversation.ConversationID(m.ConversationID)).Only(r.ctx)
	switch {
	case err == nil:
		upd := r.tx.Conversation.UpdateOne(c).SetLastSeenImportID(r.id)
		if m.Title != "" {
			upd.SetTitle(m.Title)
		}
		err = upd.Exec(r.ctx)
		r.count("conversations", false)
	case ent.IsNotFound(err):
		c, err = r.tx.Conversation.Create().
			SetConversationID(m.ConversationID).
			SetTitle(m.Title).
			SetSourceFile(m.SourceFile).
			SetFirstSeenImportID(r.id).
			SetLastSeenImportID(r.id).
			Save(r.ctx)
		r.count("conversations", true)
	}
	if err != nil {
		return 0, fmt.Errorf("conversation %s: %w", m.ConversationID, err)
	}
	r.convs[m.ConversationID] = c.ID
	return c.ID, nil
}

func (r *run) messages(a *archive.Archive) error {
	ms, err := a.Messages()
	if err != nil {
		return err
	}
	for _, f := range a.MessageFiles() {
		r.handled[f] = true
	}
	for _, m := range ms {
		existing, err := r.tx.Message.Query().Where(message.Key(m.Key)).Only(r.ctx)
		if err == nil {
			// Folder can change between exports (e.g. inbox → archive).
			if err := r.tx.Message.UpdateOne(existing).SetLastSeenImportID(r.id).SetFolder(m.Folder).Exec(r.ctx); err != nil {
				return fmt.Errorf("message %s: %w", m.Key, err)
			}
			r.count("messages", false)
			continue
		}
		if !ent.IsNotFound(err) {
			return err
		}
		convID, err := r.conversation(m)
		if err != nil {
			return err
		}
		senderID, err := r.person(m.SenderURL, m.From)
		if err != nil {
			return err
		}
		recipientIDs, err := r.recipients(m)
		if err != nil {
			return err
		}
		c := r.tx.Message.Create().
			SetKey(m.Key).
			SetNillableSentAt(timePtr(m.SentAt)).
			SetFromName(m.From).
			SetSenderURL(m.SenderURL).
			SetToNames(m.To).
			SetRecipientUrls(m.RecipientURLs).
			SetSubject(m.Subject).
			SetContent(m.Content).
			SetBodyText(m.BodyText).
			SetFolder(m.Folder).
			SetAttachments(m.Attachments).
			SetSourceFile(m.SourceFile).
			AddRecipientIDs(recipientIDs...).
			SetFirstSeenImportID(r.id).
			SetLastSeenImportID(r.id)
		if convID != 0 {
			c.SetConversationID(convID)
		}
		if senderID != 0 {
			c.SetSenderID(senderID)
		}
		if err := c.Exec(r.ctx); err != nil {
			return fmt.Errorf("message %s: %w", m.Key, err)
		}
		r.count("messages", true)
	}
	return nil
}

// recipients resolves recipient people from profile URLs. Names are attached
// only when the TO list splits into exactly as many names as URLs, since
// names may themselves contain commas.
func (r *run) recipients(m archive.Message) ([]int, error) {
	names := strings.Split(m.To, ",")
	useNames := len(names) == len(m.RecipientURLs)
	ids := make([]int, 0, len(m.RecipientURLs))
	seen := map[int]bool{}
	for i, u := range m.RecipientURLs {
		name := ""
		if useNames {
			name = strings.TrimSpace(names[i])
		}
		id, err := r.person(u, name)
		if err != nil {
			return nil, err
		}
		if id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *run) invitations(a *archive.Archive) error {
	invs, err := a.Invitations()
	if err != nil {
		return err
	}
	if f := a.Find("Invitations.csv"); f != "" {
		r.handled[f] = true
	}
	for _, inv := range invs {
		n, err := r.tx.Invitation.Update().Where(invitation.Key(inv.Key)).SetLastSeenImportID(r.id).Save(r.ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			r.count("invitations", false)
			continue
		}
		err = r.tx.Invitation.Create().
			SetKey(inv.Key).
			SetFromName(inv.From).
			SetToName(inv.To).
			SetNillableSentAt(timePtr(inv.SentAt)).
			SetMessage(inv.Message).
			SetDirection(inv.Direction).
			SetInviterURL(inv.InviterURL).
			SetInviteeURL(inv.InviteeURL).
			SetFirstSeenImportID(r.id).
			SetLastSeenImportID(r.id).
			Exec(r.ctx)
		if err != nil {
			return fmt.Errorf("invitation %s: %w", inv.Key, err)
		}
		r.count("invitations", true)
	}
	return nil
}

func (r *run) savedItems(a *archive.Archive) error {
	items, err := a.SavedItems()
	if err != nil {
		return err
	}
	for _, f := range a.SavedItemFiles() {
		r.handled[f] = true
	}
	for _, it := range items {
		n, err := r.tx.SavedItem.Update().Where(saveditem.Key(it.Key)).SetLastSeenImportID(r.id).Save(r.ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			r.count("saved_items", false)
			continue
		}
		err = r.tx.SavedItem.Create().
			SetKey(it.Key).
			SetURL(it.URL).
			SetNillableSavedAt(timePtr(it.SavedAt)).
			SetTitle(it.Title).
			SetAuthor(it.Author).
			SetContent(it.Content).
			SetExtra(it.Extra).
			SetSourceFile(it.SourceFile).
			SetFirstSeenImportID(r.id).
			SetLastSeenImportID(r.id).
			Exec(r.ctx)
		if err != nil {
			return fmt.Errorf("saved item %s: %w", it.Key, err)
		}
		r.count("saved_items", true)
	}
	return nil
}

// rawRecords stores every row of files not handled by a typed importer.
func (r *run) rawRecords(a *archive.Archive) error {
	for _, f := range a.Files() {
		if r.handled[f] {
			continue
		}
		t, err := a.ReadCSV(f)
		if err != nil {
			return err
		}
		file := strings.TrimPrefix(path.Clean(f), "./")
		occurrences := map[string]int{}
		for _, row := range t.Rows {
			// Identical rows are kept distinct by occurrence index.
			h := rowHash(t.Header, row)
			occurrences[h]++
			if n := occurrences[h]; n > 1 {
				h += "-" + strconv.Itoa(n)
			}
			n, err := r.tx.RawRecord.Update().
				Where(rawrecord.File(file), rawrecord.RowHash(h)).
				SetLastSeenImportID(r.id).Save(r.ctx)
			if err != nil {
				return err
			}
			if n > 0 {
				r.count("raw:"+file, false)
				continue
			}
			err = r.tx.RawRecord.Create().
				SetFile(file).
				SetRowHash(h).
				SetData(row).
				SetFirstSeenImportID(r.id).
				SetLastSeenImportID(r.id).
				Exec(r.ctx)
			if err != nil {
				return fmt.Errorf("raw record %s: %w", file, err)
			}
			r.count("raw:"+file, true)
		}
	}
	return nil
}

func rowHash(header []string, row archive.Row) string {
	keys := append([]string(nil), header...)
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		parts = append(parts, k, row[k])
	}
	return archive.HashKey(parts...)
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
