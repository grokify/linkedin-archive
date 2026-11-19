package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// SavedItem is a saved post or item (Complete export).
type SavedItem struct {
	ent.Schema
}

// Mixin of the SavedItem.
func (SavedItem) Mixin() []ent.Mixin { return []ent.Mixin{SeenMixin{}} }

// Fields of the SavedItem.
func (SavedItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").Unique(),
		field.Text("url").Optional(),
		field.Time("saved_at").Optional().Nillable(),
		field.Text("title").Optional(),
		field.String("author").Optional(),
		field.Text("content").Optional(),
		field.JSON("extra", map[string]string{}).Optional(),
		field.String("source_file").Optional(),
	}
}
