package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// ImportRun records one imported export archive.
type ImportRun struct {
	ent.Schema
}

// Fields of the ImportRun.
func (ImportRun) Fields() []ent.Field {
	return []ent.Field{
		field.String("sha256").Unique().Immutable(),
		field.String("source_name"),
		field.String("export_kind"),
		field.Time("imported_at").Default(time.Now).Immutable(),
		field.JSON("files", []string{}).Optional(),
		field.JSON("stats", map[string]FileStats{}).Optional(),
	}
}

// FileStats counts rows for one entity or file in an import.
type FileStats struct {
	Seen     int `json:"seen"`
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped,omitempty"` // rows with no usable identity
}
