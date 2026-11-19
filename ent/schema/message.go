package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Message is one message in a conversation.
type Message struct {
	ent.Schema
}

// Mixin of the Message.
func (Message) Mixin() []ent.Mixin { return []ent.Mixin{SeenMixin{}} }

// Fields of the Message.
func (Message) Fields() []ent.Field {
	text := map[string]string{dialect.MySQL: "longtext"}
	return []ent.Field{
		field.String("key").Unique().Comment("Content hash; stable across exports"),
		field.Time("sent_at").Optional().Nillable(),
		field.String("from_name").Default(""),
		field.String("sender_url").Optional(),
		field.Text("to_names").Optional(),
		field.Strings("recipient_urls").Optional(),
		field.String("subject").Default(""),
		field.Text("content").Default("").SchemaType(text).Comment("Raw content, may contain HTML"),
		field.Text("body_text").Default("").SchemaType(text).Comment("Plain text used for search"),
		field.String("folder").Optional(),
		field.Text("attachments").Optional(),
		field.String("source_file").Optional(),
		field.Int("conversation_id").Optional(),
		field.Int("sender_id").Optional(),
	}
}

// Edges of the Message.
func (Message) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", Conversation.Type).Ref("messages").Field("conversation_id").Unique(),
		edge.From("sender", Person.Type).Ref("sent_messages").Field("sender_id").Unique(),
		edge.To("recipients", Person.Type),
	}
}

// Indexes of the Message.
func (Message) Indexes() []ent.Index {
	return []ent.Index{index.Fields("sent_at"), index.Fields("folder")}
}
