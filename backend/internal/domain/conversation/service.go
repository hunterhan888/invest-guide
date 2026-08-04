package conversation

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/invest-guide/backend/internal/platform/response"
)

type Service struct {
	convs     ConversationRepository
	msgs      MessageRepository
	assistant *assistant.Service
}

func NewService(convs ConversationRepository, msgs MessageRepository, asst *assistant.Service) *Service {
	return &Service{convs: convs, msgs: msgs, assistant: asst}
}

func (s *Service) CreateConversation(ctx context.Context, userID string, req CreateConversationRequest) (*ConversationDTO, error) {
	title := req.Title
	if title == "" {
		title = "新会话"
	}
	conv := &Conversation{
		ID:      uuid.NewString(),
		UserID:  userID,
		Title:   title,
		Country: req.Country,
	}
	if err := s.convs.Create(ctx, conv); err != nil {
		return nil, err
	}
	dto := conv.ToDTO()
	return &dto, nil
}

func (s *Service) GetConversation(ctx context.Context, id, userID string) (*ConversationDTO, error) {
	conv, err := s.convs.Get(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	dto := conv.ToDTO()
	return &dto, nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, page, pageSize int) ([]*ConversationDTO, int64, error) {
	items, total, err := s.convs.ListByUser(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*ConversationDTO, len(items))
	for i, c := range items {
		dto := c.ToDTO()
		dtos[i] = &dto
	}
	return dtos, total, nil
}

func (s *Service) DeleteConversation(ctx context.Context, id, userID string) error {
	return s.convs.Delete(ctx, id, userID)
}

func (s *Service) ListMessages(ctx context.Context, convID, userID string, page, pageSize int) ([]*MessageDTO, int64, error) {
	if _, err := s.convs.Get(ctx, convID, userID); err != nil {
		return nil, 0, err
	}
	items, total, err := s.msgs.ListByConversation(ctx, convID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*MessageDTO, len(items))
	for i, m := range items {
		dto := m.ToDTO()
		dtos[i] = &dto
	}
	return dtos, total, nil
}

// PostMessage 创建 user 消息 + 占位 assistant 消息，返回 assistant message ID 供 stream 订阅
func (s *Service) PostMessage(ctx context.Context, convID, userID, content string) (*PostMessageResponse, error) {
	conv, err := s.convs.Get(ctx, convID, userID)
	if err != nil {
		return nil, err
	}

	userMsg := &Message{
		ID:             uuid.NewString(),
		ConversationID: convID,
		Role:           RoleUser,
		Content:        content,
	}
	if err := s.msgs.Create(ctx, userMsg); err != nil {
		return nil, err
	}

	assistantMsg := &Message{
		ID:             uuid.NewString(),
		ConversationID: convID,
		Role:           RoleAssistant,
		Content:        "",
	}
	if err := s.msgs.Create(ctx, assistantMsg); err != nil {
		return nil, err
	}

	if conv.Title == "新会话" {
		title := content
		if r := []rune(title); len(r) > 20 {
			title = string(r[:20]) + "..."
		}
		_ = s.convs.Update(ctx, convID, userID, UpdateConversationParams{Title: &title})
	}

	return &PostMessageResponse{MessageID: assistantMsg.ID}, nil
}

// StreamAnswer 由 SSE handler 调用：执行 RAG + LLM 流式。
// 返回 sources（供 SSE 在首次 message 之前发送）和 delta channel。
func (s *Service) StreamAnswer(ctx context.Context, convID, userID, assistantMessageID string) (
	sources []assistant.ContextSource,
	ch <-chan llm.StreamChunk,
	err error,
) {
	conv, err := s.convs.Get(ctx, convID, userID)
	if err != nil {
		return nil, nil, err
	}

	// 确认 assistant 占位消息存在
	target, err := s.msgs.Get(ctx, assistantMessageID)
	if err != nil {
		return nil, nil, response.ErrNotFound
	}
	if target.Role != RoleAssistant {
		return nil, nil, response.ErrInvalidInput
	}

	msgs, _, err := s.msgs.ListByConversation(ctx, convID, 1, 20)
	if err != nil {
		return nil, nil, err
	}
	var userQuery string
	for _, m := range msgs {
		if m.ID == assistantMessageID {
			break
		}
		if m.Role == RoleUser {
			userQuery = m.Content
		}
	}
	if userQuery == "" {
		return nil, nil, response.ErrInvalidInput
	}

	ch, sources, err = s.assistant.Stream(ctx, userQuery, conv.Country)
	if err != nil {
		return nil, nil, response.ErrBadGateway
	}
	return sources, ch, nil
}

// FinalizeAnswer 流结束后持久化 assistant 消息内容与引用源。
// 使用独立 context，确保即便客户端在 done 后立即断连也能落库。
func (s *Service) FinalizeAnswer(assistantMessageID, content string, sources []assistant.ContextSource, tokens int) error {
	ctx := context.Background()
	srcsJSON, _ := json.Marshal(sources)
	return s.msgs.Update(ctx, assistantMessageID, UpdateMessageParams{
		Content:    &content,
		Sources:    srcsJSON,
		TokensUsed: &tokens,
	})
}
