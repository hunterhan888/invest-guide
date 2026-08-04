package conversation

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	items, total, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":   items,
			"total":   total,
			"hasMore": int64(page*pageSize) < total,
		},
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidInput)
		return
	}
	dto, err := h.service.CreateConversation(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusCreated, dto)
}

func (h *Handler) Get(c *gin.Context) {
	dto, err := h.service.GetConversation(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusOK, dto)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.DeleteConversation(c.Request.Context(), c.Param("id"), c.GetString("userID")); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListMessages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	items, total, err := h.service.ListMessages(c.Request.Context(), c.Param("id"), c.GetString("userID"), page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":   items,
			"total":   total,
			"hasMore": int64(page*pageSize) < total,
		},
	})
}

func (h *Handler) PostMessage(c *gin.Context) {
	var req PostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidInput)
		return
	}
	resp, err := h.service.PostMessage(c.Request.Context(), c.Param("id"), c.GetString("userID"), req.Content)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusCreated, resp)
}

// Stream 是 SSE 流式回答端点
// 路由：GET /conversations/:id/messages/:messageId/stream
func (h *Handler) Stream(c *gin.Context) {
	convID := c.Param("id")
	messageID := c.Param("messageId")
	userID := c.GetString("userID")

	sources, ch, err := h.service.StreamAnswer(c.Request.Context(), convID, userID, messageID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	sse := NewSSEWriter(c.Writer)

	if len(sources) > 0 {
		chunks := make([]map[string]string, len(sources))
		for i, s := range sources {
			chunks[i] = map[string]string{"id": s.ChunkID, "title": s.Title, "snippet": s.Snippet}
		}
		if err := sse.Send("sources", map[string]interface{}{"chunks": chunks}); err != nil {
			return
		}
	}

	var contentBuilder strings.Builder
	var tokensUsed int

	for {
		select {
		case <-c.Request.Context().Done():
			// 客户端断连：把已生成的部分内容与来源落库，避免占位消息永远为空
			_ = h.service.FinalizeAnswer(messageID, contentBuilder.String(), sources, tokensUsed)
			return
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			if chunk.Err != nil {
				_ = sse.Send("error", map[string]string{
					"code":    "LLM_ERROR",
					"message": "stream failed",
				})
				// 落库已生成的部分内容与来源，避免占位消息永远为空
				_ = h.service.FinalizeAnswer(messageID, contentBuilder.String(), sources, tokensUsed)
				return
			}
			if chunk.Done {
				tokensUsed = chunk.TokensUsed
				_ = sse.Send("done", map[string]interface{}{
					"messageId":  messageID,
					"tokensUsed": tokensUsed,
				})
				_ = h.service.FinalizeAnswer(messageID, contentBuilder.String(), sources, tokensUsed)
				return
			}
			if chunk.Delta != "" {
				contentBuilder.WriteString(chunk.Delta)
				_ = sse.Send("message", map[string]string{"delta": chunk.Delta})
			}
			sse.MaybeHeartbeat()
		}
	}
}
