package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/platform/response"
)

type ModelsHandler struct {
	llmModel       string
	embeddingModel string
}

func NewModelsHandler(llmModel, embeddingModel string) *ModelsHandler {
	return &ModelsHandler{llmModel: llmModel, embeddingModel: embeddingModel}
}

func (h *ModelsHandler) Models(c *gin.Context) {
	response.Ok(c, http.StatusOK, gin.H{
		"llm":       h.llmModel,
		"embedding": h.embeddingModel,
	})
}
