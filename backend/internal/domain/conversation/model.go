package conversation

import (
	"encoding/json"
	"time"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Conversation struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"not null;index" json:"userId"`
	Title     string    `json:"title"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Conversation) TableName() string { return "conversations" }

type Message struct {
	ID             string          `gorm:"primaryKey" json:"id"`
	ConversationID string          `gorm:"not null;index" json:"conversationId"`
	Role           string          `gorm:"not null" json:"role"`
	Content        string          `gorm:"not null;type:text" json:"content"`
	Sources        json.RawMessage `gorm:"type:jsonb" json:"sources,omitempty"`
	TokensUsed     int             `gorm:"not null;default:0" json:"tokensUsed"`
	CreatedAt      time.Time       `json:"createdAt"`
}

func (Message) TableName() string { return "messages" }

// DTOs
type CreateConversationRequest struct {
	Title   string `json:"title,omitempty" binding:"max=200"`
	Country string `json:"country,omitempty" binding:"max=100"`
}

type ConversationDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (c *Conversation) ToDTO() ConversationDTO {
	return ConversationDTO{ID: c.ID, Title: c.Title, Country: c.Country,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

type PostMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=10000"`
}

type PostMessageResponse struct {
	MessageID string `json:"messageId"`
}

type MessageDTO struct {
	ID         string          `json:"id"`
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	Sources    json.RawMessage `json:"sources,omitempty"`
	TokensUsed int             `json:"tokensUsed"`
	CreatedAt  time.Time       `json:"createdAt"`
}

func (m *Message) ToDTO() MessageDTO {
	return MessageDTO{
		ID: m.ID, Role: m.Role, Content: m.Content,
		Sources: m.Sources, TokensUsed: m.TokensUsed, CreatedAt: m.CreatedAt,
	}
}
