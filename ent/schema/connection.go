package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Connection is a first-degree connection from Connections.csv.
type Connection struct {
	ent.Schema
}

// Mixin of the Connection.
func (Connection) Mixin() []ent.Mixin { return []ent.Mixin{SeenMixin{}} }

// Fields of the Connection.
func (Connection) Fields() []ent.Field {
	return []ent.Field{
		field.String("first_name").Optional(),
		field.String("last_name").Optional(),
		field.String("profile_url").Optional(),
		field.String("email").Optional(),
		field.String("company").Optional(),
		field.String("position").Optional(),
		field.Time("connected_on").Optional().Nillable(),
		field.Int("person_id").Unique(),
	}
}

// Edges of the Connection.
func (Connection) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("person", Person.Type).Ref("connection").Field("person_id").Unique().Required(),
	}
}

// Indexes of the Connection.
func (Connection) Indexes() []ent.Index {
	return []ent.Index{index.Fields("company")}
}
