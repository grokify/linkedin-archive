package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Person is anyone appearing in the archive, keyed by profile URL when known.
type Person struct {
	ent.Schema
}

// Fields of the Person.
func (Person) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").Unique().Comment("Normalized profile URL, or name:<lowercased name>"),
		field.String("profile_url").Optional(),
		field.String("name").Optional(),
	}
}

// Edges of the Person.
func (Person) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sent_messages", Message.Type),
		edge.From("received_messages", Message.Type).Ref("recipients"),
		edge.To("connection", Connection.Type).Unique(),
	}
}
