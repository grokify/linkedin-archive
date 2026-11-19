package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Conversation is a message thread identified by LinkedIn's conversation ID.
type Conversation struct {
	ent.Schema
}

// Mixin of the Conversation.
func (Conversation) Mixin() []ent.Mixin { return []ent.Mixin{SeenMixin{}} }

// Fields of the Conversation.
func (Conversation) Fields() []ent.Field {
	return []ent.Field{
		field.String("conversation_id").Unique(),
		field.String("title").Optional(),
		field.String("source_file").Optional(),
	}
}

// Edges of the Conversation.
func (Conversation) Edges() []ent.Edge {
	return []ent.Edge{edge.To("messages", Message.Type)}
}
