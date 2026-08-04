package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Health(c *gin.Context) {
	response.Ok(c, http.StatusOK, h.service.Health())
}

func (h *Handler) Version(c *gin.Context) {
	response.Ok(c, http.StatusOK, h.service.Version())
}
