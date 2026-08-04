package conversation

import (
	"context"
	"testing"

	"github.com/invest-guide/backend/internal/platform/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGORMRepo_ConversationUpdate(t *testing.T) {
	db := newTestDB(t)
	repo := NewGORMConversationRepository(db)

	conv := &Conversation{ID: "c-1", UserID: "u-1", Title: "旧"}
	require.NoError(t, repo.Create(context.Background(), conv))

	newTitle := "新标题"
	require.NoError(t, repo.Update(context.Background(), "c-1", "u-1", UpdateConversationParams{Title: &newTitle}))

	got, err := repo.Get(context.Background(), "c-1", "u-1")
	require.NoError(t, err)
	assert.Equal(t, "新标题", got.Title)

	// 其他用户 update → not found
	err = repo.Update(context.Background(), "c-1", "u-other", UpdateConversationParams{Title: &newTitle})
	assert.ErrorIs(t, err, response.ErrNotFound)
}

func TestGORMRepo_MessageGetAndUpdate(t *testing.T) {
	db := newTestDB(t)
	repo := NewGORMMessageRepository(db)

	msg := &Message{ID: "m-1", ConversationID: "c-1", Role: RoleAssistant, Content: ""}
	require.NoError(t, repo.Create(context.Background(), msg))

	content := "回答内容"
	tokens := 42
	require.NoError(t, repo.Update(context.Background(), "m-1", UpdateMessageParams{
		Content: &content, TokensUsed: &tokens,
	}))

	got, err := repo.Get(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, "回答内容", got.Content)
	assert.Equal(t, 42, got.TokensUsed)
}

func TestGORMRepo_MessageNotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewGORMMessageRepository(db)
	_, err := repo.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, response.ErrNotFound)
}
