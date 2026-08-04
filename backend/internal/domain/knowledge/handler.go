package knowledge

import (
	"net/http"
	"strconv"

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
	country := c.Query("country")

	docs, total, err := h.service.List(c.Request.Context(), page, pageSize, country)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":   docs,
			"total":   total,
			"hasMore": int64(page*pageSize) < total,
		},
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateDocRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidInput)
		return
	}
	dto, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusAccepted, dto)
}

func (h *Handler) Get(c *gin.Context) {
	dto, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusOK, dto)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Retry(c *gin.Context) {
	if err := h.service.Retry(c.Request.Context(), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusAccepted, nil)
}
