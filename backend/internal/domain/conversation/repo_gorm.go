package conversation

import (
	"context"
	"errors"

	"github.com/invest-guide/backend/internal/platform/response"
	"gorm.io/gorm"
)

type gormConversationRepository struct {
	db *gorm.DB
}

func NewGORMConversationRepository(db *gorm.DB) ConversationRepository {
	return &gormConversationRepository{db: db}
}

func (r *gormConversationRepository) Create(ctx context.Context, c *Conversation) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *gormConversationRepository) Get(ctx context.Context, id, userID string) (*Conversation, error) {
	var c Conversation
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *gormConversationRepository) Update(ctx context.Context, id, userID string, params UpdateConversationParams) error {
	updates := map[string]interface{}{}
	if params.Title != nil {
		updates["title"] = *params.Title
	}
	if params.Country != nil {
		updates["country"] = *params.Country
	}
	if len(updates) == 0 {
		return nil
	}
	res := r.db.WithContext(ctx).Model(&Conversation{}).
		Where("id = ? AND user_id = ?", id, userID).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return response.ErrNotFound
	}
	return nil
}

func (r *gormConversationRepository) Delete(ctx context.Context, id, userID string) error {
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&Conversation{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return response.ErrNotFound
	}
	return nil
}

func (r *gormConversationRepository) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]*Conversation, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	q := r.db.WithContext(ctx).Model(&Conversation{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*Conversation
	if err := q.Order("updated_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type gormMessageRepository struct {
	db *gorm.DB
}

func NewGORMMessageRepository(db *gorm.DB) MessageRepository {
	return &gormMessageRepository{db: db}
}

func (r *gormMessageRepository) Create(ctx context.Context, m *Message) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *gormMessageRepository) Get(ctx context.Context, id string) (*Message, error) {
	var m Message
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *gormMessageRepository) ListByConversation(ctx context.Context, convID string, page, pageSize int) ([]*Message, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	q := r.db.WithContext(ctx).Model(&Message{}).Where("conversation_id = ?", convID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*Message
	if err := q.Order("created_at ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *gormMessageRepository) Update(ctx context.Context, id string, params UpdateMessageParams) error {
	updates := map[string]interface{}{}
	if params.Content != nil {
		updates["content"] = *params.Content
	}
	if params.Sources != nil {
		updates["sources"] = params.Sources
	}
	if params.TokensUsed != nil {
		updates["tokens_used"] = *params.TokensUsed
	}
	if len(updates) == 0 {
		return nil
	}
	res := r.db.WithContext(ctx).Model(&Message{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	return nil
}
