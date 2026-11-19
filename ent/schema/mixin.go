// Package schema defines the Ent schemas for linkedin-archive.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// SeenMixin records the first and most recent import containing a record.
type SeenMixin struct {
	mixin.Schema
}

// Fields of the SeenMixin.
func (SeenMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Int("first_seen_import_id").Comment("Import that first contained this record"),
		field.Int("last_seen_import_id").Comment("Most recent import that contained this record"),
	}
}
