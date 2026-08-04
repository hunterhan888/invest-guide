package conversation

import "github.com/gin-gonic/gin"

func Register(group *gin.RouterGroup, h *Handler) {
	convs := group.Group("/conversations")
	convs.GET("", h.List)
	convs.POST("", h.Create)
	convs.GET("/:id", h.Get)
	convs.DELETE("/:id", h.Delete)
	convs.GET("/:id/messages", h.ListMessages)
	convs.POST("/:id/messages", h.PostMessage)
	convs.GET("/:id/messages/:messageId/stream", h.Stream)
}
