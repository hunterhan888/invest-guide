package conversation

import (
	"context"
	"testing"

	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeSearcherForConv 实现 assistant.KnowledgeSearcher
type fakeSearcherForConv struct{}

func (fakeSearcherForConv) Search(ctx context.Context, query, country string, topK int) ([]assistant.ContextSource, error) {
	return nil, nil
}

func newConvService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	asst := assistant.NewService(llm.NewFakeProvider("", []string{"Hi"}, 1), fakeSearcherForConv{})
	svc := NewService(
		NewGORMConversationRepository(db),
		NewGORMMessageRepository(db),
		asst,
	)
	return svc, db
}

func TestService_ConversationCRUD(t *testing.T) {
	svc, _ := newConvService(t)
	ctx := context.Background()

	dto, err := svc.CreateConversation(ctx, "u-1", CreateConversationRequest{Title: "测试", Country: "越南"})
	require.NoError(t, err)
	assert.Equal(t, "测试", dto.Title)
	assert.Equal(t, "越南", dto.Country)

	got, err := svc.GetConversation(ctx, dto.ID, "u-1")
	require.NoError(t, err)
	assert.Equal(t, dto.ID, got.ID)

	// 其他用户访问 → 404
	_, err = svc.GetConversation(ctx, dto.ID, "u-other")
	assert.Error(t, err)

	items, total, err := svc.ListConversations(ctx, "u-1", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, items, 1)

	require.NoError(t, svc.DeleteConversation(ctx, dto.ID, "u-1"))
	_, err = svc.GetConversation(ctx, dto.ID, "u-1")
	assert.Error(t, err)
}

func TestService_CreateConversation_DefaultTitle(t *testing.T) {
	svc, _ := newConvService(t)
	dto, err := svc.CreateConversation(context.Background(), "u-1", CreateConversationRequest{})
	require.NoError(t, err)
	assert.Equal(t, "新会话", dto.Title)
}

func TestService_PostMessageAndList(t *testing.T) {
	svc, _ := newConvService(t)
	ctx := context.Background()

	conv, _ := svc.CreateConversation(ctx, "u-1", CreateConversationRequest{})
	resp, err := svc.PostMessage(ctx, conv.ID, "u-1", "越南税收多少？")
	require.NoError(t, err)
	require.NotEmpty(t, resp.MessageID)

	items, total, err := svc.ListMessages(ctx, conv.ID, "u-1", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total) // user + assistant
	assert.Len(t, items, 2)
	assert.Equal(t, RoleUser, items[0].Role)
	assert.Equal(t, RoleAssistant, items[1].Role)
}

func TestService_PostMessage_WrongUser(t *testing.T) {
	svc, _ := newConvService(t)
	ctx := context.Background()

	conv, _ := svc.CreateConversation(ctx, "u-1", CreateConversationRequest{})
	_, err := svc.PostMessage(ctx, conv.ID, "u-other", "hi")
	assert.Error(t, err)
}

func TestService_StreamAnswer_AndFinalize(t *testing.T) {
	svc, _ := newConvService(t)
	ctx := context.Background()

	conv, _ := svc.CreateConversation(ctx, "u-1", CreateConversationRequest{Country: "越南"})
	resp, _ := svc.PostMessage(ctx, conv.ID, "u-1", "越南企业所得税？")

	sources, ch, err := svc.StreamAnswer(ctx, conv.ID, "u-1", resp.MessageID)
	require.NoError(t, err)
	assert.Empty(t, sources) // fakeSearcher 返回空

	var content string
	for c := range ch {
		if !c.Done {
			content += c.Delta
		}
	}
	assert.Equal(t, "Hi", content)

	require.NoError(t, svc.FinalizeAnswer(resp.MessageID, content, sources, 1))

	// 校验消息已持久化
	items, _, _ := svc.ListMessages(ctx, conv.ID, "u-1", 1, 50)
	assert.Equal(t, "Hi", items[1].Content)
	assert.Equal(t, 1, items[1].TokensUsed)
}

func TestService_StreamAnswer_InvalidMessage(t *testing.T) {
	svc, _ := newConvService(t)
	ctx := context.Background()

	conv, _ := svc.CreateConversation(ctx, "u-1", CreateConversationRequest{})
	_, _, err := svc.StreamAnswer(ctx, conv.ID, "u-1", "nonexistent")
	assert.Error(t, err)
}
