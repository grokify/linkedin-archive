package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/linkedin-archive/archive"
	"github.com/grokify/linkedin-archive/ent/importrun"
	"github.com/grokify/linkedin-archive/ent/saveditem"
	"github.com/grokify/linkedin-archive/ent/schema"
	"github.com/grokify/linkedin-archive/importer"
	"github.com/grokify/linkedin-archive/search"
	"github.com/grokify/linkedin-archive/search/engine"
	"github.com/grokify/linkedin-archive/store"
)

type rootOpts struct {
	dsn string
}

func newRootCmd() *cobra.Command {
	o := &rootOpts{}
	cmd := &cobra.Command{
		Use:           "liarchive",
		Short:         "Import and search LinkedIn data export archives",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	def := os.Getenv("LIARCHIVE_DB")
	if def == "" {
		def = store.DefaultDSN
	}
	cmd.PersistentFlags().StringVar(&o.dsn, "db", def, "database DSN: sqlite:<path> or postgres://… (env LIARCHIVE_DB)")
	cmd.AddCommand(newImportCmd(o), newSearchCmd(o), newImportsCmd(o), newStatsCmd(o), newSavedCmd(o))
	return cmd
}

// withStore opens the store, runs fn and closes the store.
func withStore(ctx context.Context, o *rootOpts, fn func(*store.Store) error) (err error) {
	s, err := store.Open(ctx, o.dsn)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := s.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()
	return fn(s)
}

func newImportCmd(o *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "import <export.zip|dir>...",
		Short: "Import one or more LinkedIn export archives",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return withStore(ctx, o, func(s *store.Store) error {
				for _, p := range args {
					if err := importOne(ctx, cmd.OutOrStdout(), s, p); err != nil {
						return fmt.Errorf("%s: %w", p, err)
					}
				}
				// Ensure the search index exists (it is kept in sync afterwards).
				_, err := engine.New(ctx, s)
				return err
			})
		},
	}
}

func importOne(ctx context.Context, w io.Writer, s *store.Store, p string) (err error) {
	a, err := archive.Open(p)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := a.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()
	start := time.Now()
	res, err := importer.Import(ctx, s.Client, a)
	if err != nil {
		return err
	}
	if res.AlreadyImported {
		_, err = fmt.Fprintf(w, "%s: already imported (import #%d)\n", a.Name(), res.ImportID)
		return err
	}
	if _, err := fmt.Fprintf(w, "%s: import #%d (%s export) in %s\n", a.Name(), res.ImportID, res.Kind, time.Since(start).Round(time.Millisecond)); err != nil {
		return err
	}
	return printStats(w, res.Stats)
}

func printStats(w io.Writer, stats map[string]schema.FileStats) error {
	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		// Typed entities first, raw files after.
		ri, rj := strings.HasPrefix(keys[i], "raw:"), strings.HasPrefix(keys[j], "raw:")
		if ri != rj {
			return rj
		}
		return keys[i] < keys[j]
	})
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "  ENTITY\tSEEN\tNEW\tSKIPPED"); err != nil {
		return err
	}
	for _, k := range keys {
		if _, err := fmt.Fprintf(tw, "  %s\t%d\t%d\t%d\n", k, stats[k].Seen, stats[k].Inserted, stats[k].Skipped); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func newSearchCmd(o *rootOpts) *cobra.Command {
	var (
		q            search.Query
		since, until string
		asJSON, full bool
	)
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Full-text search messages",
		Long: `Full-text search messages across all folders.

Query syntax: words are ANDed; "quoted phrases"; OR between terms;
-word to exclude; word* for prefix matches; (parentheses) to group.
AND binds tighter than OR.

Examples:
  liarchive search 'acme (equity OR shares OR stock)' --since 2025-01-01
  liarchive search '"secondary sale"' --folder ARCHIVE
  liarchive search --from "Jane Doe"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			q.Text = strings.Join(args, " ")
			var err error
			if q.Since, err = parseDate(since); err != nil {
				return fmt.Errorf("--since: %w", err)
			}
			if q.Until, err = parseDate(until); err != nil {
				return fmt.Errorf("--until: %w", err)
			}
			if q.Text == "" && q.From == "" && q.Folder == "" && q.Since.IsZero() && q.Until.IsZero() {
				return errors.New("provide a query or at least one filter")
			}
			ctx := cmd.Context()
			return withStore(ctx, o, func(s *store.Store) error {
				srch, err := engine.New(ctx, s)
				if err != nil {
					return err
				}
				results, err := srch.Search(ctx, q)
				if err != nil {
					return err
				}
				return printResults(ctx, cmd.OutOrStdout(), results, asJSON, full, s)
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&q.From, "from", "", "sender name or profile URL substring")
	f.StringVar(&q.Folder, "folder", "", "folder: INBOX, ARCHIVE, SPAM (default all)")
	f.StringVar(&since, "since", "", "only messages on or after this date (YYYY-MM-DD)")
	f.StringVar(&until, "until", "", "only messages before this date (YYYY-MM-DD)")
	f.IntVarP(&q.Limit, "limit", "n", search.DefaultLimit, "maximum results")
	f.BoolVar(&asJSON, "json", false, "output JSON")
	f.BoolVar(&full, "full", false, "print full message text instead of a snippet")
	return cmd
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", s)
}

func printResults(ctx context.Context, w io.Writer, results []search.Result, asJSON, full bool, s *store.Store) error {
	if full {
		for i := range results {
			m, err := s.Client.Message.Get(ctx, results[i].MessageID)
			if err != nil {
				return err
			}
			results[i].Snippet = m.BodyText
		}
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if results == nil {
			results = []search.Result{}
		}
		return enc.Encode(results)
	}
	for _, r := range results {
		date := "unknown date"
		if r.SentAt != nil {
			date = r.SentAt.Format("2006-01-02 15:04")
		}
		header := fmt.Sprintf("%s  %s  [%s]", date, r.From, r.Folder)
		if r.SenderURL != "" {
			header += "  " + r.SenderURL
		}
		if _, err := fmt.Fprintln(w, header); err != nil {
			return err
		}
		if r.Subject != "" {
			if _, err := fmt.Fprintf(w, "  Subject: %s\n", r.Subject); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "  %s\n  conversation: %s\n\n", r.Snippet, r.ConversationID); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%d result(s)\n", len(results))
	return err
}

func newImportsCmd(o *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "imports",
		Short: "List imported archives",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			return withStore(ctx, o, func(s *store.Store) error {
				runs, err := s.Client.ImportRun.Query().Order(importrun.ByID()).All(ctx)
				if err != nil {
					return err
				}
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
				if _, err := fmt.Fprintln(tw, "ID\tIMPORTED\tKIND\tSOURCE\tNEW MESSAGES\tNEW CONNECTIONS"); err != nil {
					return err
				}
				for _, r := range runs {
					if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%d\t%d\n", r.ID, r.ImportedAt.Format(time.RFC3339),
						r.ExportKind, r.SourceName, r.Stats["messages"].Inserted, r.Stats["connections"].Inserted); err != nil {
						return err
					}
				}
				return tw.Flush()
			})
		},
	}
}

func newStatsCmd(o *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show row counts per entity",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			return withStore(ctx, o, func(s *store.Store) error {
				c := s.Client
				counts := []struct {
					name string
					fn   func(context.Context) (int, error)
				}{
					{"imports", c.ImportRun.Query().Count},
					{"people", c.Person.Query().Count},
					{"connections", c.Connection.Query().Count},
					{"conversations", c.Conversation.Query().Count},
					{"messages", c.Message.Query().Count},
					{"invitations", c.Invitation.Query().Count},
					{"saved_items", c.SavedItem.Query().Count},
					{"raw_records", c.RawRecord.Query().Count},
				}
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
				for _, x := range counts {
					n, err := x.fn(ctx)
					if err != nil {
						return fmt.Errorf("count %s: %w", x.name, err)
					}
					if _, err := fmt.Fprintf(tw, "%s\t%d\n", x.name, n); err != nil {
						return err
					}
				}
				return tw.Flush()
			})
		},
	}
}

func newSavedCmd(o *rootOpts) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "saved",
		Short: "List saved items (from Complete exports)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			return withStore(ctx, o, func(s *store.Store) error {
				items, err := s.Client.SavedItem.Query().Order(saveditem.BySavedAt()).All(ctx)
				if err != nil {
					return err
				}
				w := cmd.OutOrStdout()
				if asJSON {
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(items)
				}
				if len(items) == 0 {
					_, err := fmt.Fprintln(w, "No saved items. Saved items are included in LinkedIn's Complete export, not the Basic export.")
					return err
				}
				for _, it := range items {
					date := ""
					if it.SavedAt != nil {
						date = it.SavedAt.Format("2006-01-02")
					}
					if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", date, it.URL, it.Title); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}
