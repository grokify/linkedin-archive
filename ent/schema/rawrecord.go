package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RawRecord preserves a row from a file the importer does not model.
type RawRecord struct {
	ent.Schema
}

// Mixin of the RawRecord.
func (RawRecord) Mixin() []ent.Mixin { return []ent.Mixin{SeenMixin{}} }

// Fields of the RawRecord.
func (RawRecord) Fields() []ent.Field {
	return []ent.Field{
		field.String("file"),
		field.String("row_hash"),
		field.JSON("data", map[string]string{}),
	}
}

// Indexes of the RawRecord.
func (RawRecord) Indexes() []ent.Index {
	return []ent.Index{index.Fields("file", "row_hash").Unique()}
}
