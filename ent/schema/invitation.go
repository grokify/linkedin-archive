package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Invitation is a sent or received connection invitation.
type Invitation struct {
	ent.Schema
}

// Mixin of the Invitation.
func (Invitation) Mixin() []ent.Mixin { return []ent.Mixin{SeenMixin{}} }

// Fields of the Invitation.
func (Invitation) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").Unique(),
		field.String("from_name").Optional(),
		field.String("to_name").Optional(),
		field.Time("sent_at").Optional().Nillable(),
		field.Text("message").Optional(),
		field.String("direction").Optional(),
		field.String("inviter_url").Optional(),
		field.String("invitee_url").Optional(),
	}
}
