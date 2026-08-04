package conversation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Conversation{}, &Message{}))
	return db
}

func newTestHandler(t *testing.T, userID string) (*Handler, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	svc := NewService(
		NewGORMConversationRepository(db),
		NewGORMMessageRepository(db),
		nil,
	)
	h := NewHandler(svc)
	return h, db
}

func TestHandler_Create_List_Get_Delete(t *testing.T) {
	h, _ := newTestHandler(t, "u-1")
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-1"); c.Next() })
	r.POST("/c", h.Create)
	r.GET("/c", h.List)
	r.GET("/c/:id", h.Get)
	r.DELETE("/c/:id", h.Delete)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/c", bytes.NewBufferString(`{"title":"测试","country":"越南"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var cr struct {
		Data ConversationDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	require.NotEmpty(t, cr.Data.ID)
	assert.Equal(t, "测试", cr.Data.Title)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/c", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/c/"+cr.Data.ID, nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/c/"+cr.Data.ID, nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_PostMessage_ReturnsMessageID(t *testing.T) {
	h, db := newTestHandler(t, "u-1")
	_ = db.Create(&Conversation{ID: "c-1", UserID: "u-1", Title: "x"}).Error

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-1"); c.Next() })
	r.POST("/c/:id/messages", h.PostMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/c/c-1/messages", bytes.NewBufferString(`{"content":"越南税收"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data PostMessageResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.MessageID)
}

func TestHandler_PostMessage_WrongUser_NotFound(t *testing.T) {
	h, db := newTestHandler(t, "u-1")
	_ = db.Create(&Conversation{ID: "c-1", UserID: "u-1", Title: "x"}).Error

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-other"); c.Next() })
	r.POST("/c/:id/messages", h.PostMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/c/c-1/messages", bytes.NewBufferString(`{"content":"越南税收"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ListMessages_Success(t *testing.T) {
	h, db := newTestHandler(t, "u-1")
	_ = db.Create(&Conversation{ID: "c-1", UserID: "u-1", Title: "x"}).Error
	_ = db.Create(&Message{ID: "m-1", ConversationID: "c-1", Role: RoleUser, Content: "hi"}).Error

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-1"); c.Next() })
	r.GET("/c/:id/messages", h.ListMessages)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/c/c-1/messages", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"hi"`)
}

func TestHandler_Get_WrongUser(t *testing.T) {
	h, db := newTestHandler(t, "u-1")
	_ = db.Create(&Conversation{ID: "c-1", UserID: "u-1", Title: "x"}).Error

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-other"); c.Next() })
	r.GET("/c/:id", h.Get)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/c/c-1", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// deltaThenErrProvider 先发一段 delta，再发 Err，用于模拟流式中途失败。
type deltaThenErrProvider struct{}

func (f *deltaThenErrProvider) Generate(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return &llm.GenerateResponse{Content: "x", TokensUsed: 0}, nil
}

func (f *deltaThenErrProvider) Stream(ctx context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 2)
	ch <- llm.StreamChunk{Delta: "部分回答"}
	ch <- llm.StreamChunk{Err: errors.New("boom")}
	close(ch)
	return ch, nil
}

func (f *deltaThenErrProvider) Model() string { return "fake" }

// TestHandler_Stream_Error_FinalizesPartialContent 复现"流式出错占位消息永远为空"的缺陷：
// 出错时 handler 必须把已生成的部分内容落库，而不是只发 error 事件。
func TestHandler_Stream_Error_FinalizesPartialContent(t *testing.T) {
	db := newTestDB(t)
	_ = db.Create(&Conversation{ID: "c-1", UserID: "u-1", Title: "x"}).Error
	_ = db.Create(&Message{ID: "m-user", ConversationID: "c-1", Role: RoleUser, Content: "巴基斯坦投资"}).Error
	_ = db.Create(&Message{ID: "m-asst", ConversationID: "c-1", Role: RoleAssistant, Content: ""}).Error

	asst := assistant.NewService(&deltaThenErrProvider{}, nil)
	svc := NewService(
		NewGORMConversationRepository(db),
		NewGORMMessageRepository(db),
		asst,
	)
	h := NewHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-1"); c.Next() })
	r.GET("/c/:id/messages/:messageId/stream", h.Stream)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/c/c-1/messages/m-asst/stream", nil)
	r.ServeHTTP(w, req)

	assert.Contains(t, w.Body.String(), "event: error")
	assert.Contains(t, w.Body.String(), "部分回答")

	var m Message
	require.NoError(t, db.First(&m, "id = ?", "m-asst").Error)
	assert.Equal(t, "部分回答", m.Content, "流式出错时部分内容应已落库")
}

// disconnectProvider 发一段 delta 后阻塞到 ctx 取消，模拟客户端断连。
type disconnectProvider struct {
	started chan struct{}
}

func (f *disconnectProvider) Generate(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return &llm.GenerateResponse{Content: "x", TokensUsed: 0}, nil
}

func (f *disconnectProvider) Stream(ctx context.Context, _ llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk)
	go func() {
		ch <- llm.StreamChunk{Delta: "部分回答"}
		close(f.started)
		<-ctx.Done()
	}()
	return ch, nil
}

func (f *disconnectProvider) Model() string { return "fake" }

// TestHandler_Stream_ClientDisconnect_FinalizesPartial 客户端断连时也应把已生成内容落库。
func TestHandler_Stream_ClientDisconnect_FinalizesPartial(t *testing.T) {
	db := newTestDB(t)
	_ = db.Create(&Conversation{ID: "c-1", UserID: "u-1", Title: "x"}).Error
	_ = db.Create(&Message{ID: "m-user", ConversationID: "c-1", Role: RoleUser, Content: "巴基斯坦投资"}).Error
	_ = db.Create(&Message{ID: "m-asst", ConversationID: "c-1", Role: RoleAssistant, Content: ""}).Error

	prov := &disconnectProvider{started: make(chan struct{})}
	asst := assistant.NewService(prov, nil)
	svc := NewService(
		NewGORMConversationRepository(db),
		NewGORMMessageRepository(db),
		asst,
	)
	h := NewHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-1"); c.Next() })
	r.GET("/c/:id/messages/:messageId/stream", h.Stream)

	ctx, cancel := context.WithCancel(context.Background())
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/c/c-1/messages/m-asst/stream", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		r.ServeHTTP(w, req)
		close(done)
	}()
	<-prov.started
	cancel()
	<-done

	var m Message
	require.NoError(t, db.First(&m, "id = ?", "m-asst").Error)
	assert.Equal(t, "部分回答", m.Content, "客户端断连时已生成内容应已落库")
}
