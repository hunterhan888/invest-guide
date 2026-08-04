package conversation

import "context"

type ConversationRepository interface {
	Create(ctx context.Context, c *Conversation) error
	Get(ctx context.Context, id, userID string) (*Conversation, error)
	Update(ctx context.Context, id, userID string, params UpdateConversationParams) error
	Delete(ctx context.Context, id, userID string) error
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]*Conversation, int64, error)
}

type UpdateConversationParams struct {
	Title   *string
	Country *string
}

type MessageRepository interface {
	Create(ctx context.Context, m *Message) error
	Get(ctx context.Context, id string) (*Message, error)
	ListByConversation(ctx context.Context, convID string, page, pageSize int) ([]*Message, int64, error)
	Update(ctx context.Context, id string, params UpdateMessageParams) error
}

type UpdateMessageParams struct {
	Content    *string
	Sources    []byte
	TokensUsed *int
}
