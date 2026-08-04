package knowledge

import "github.com/gin-gonic/gin"

// Register 在已鉴权的 v1 private group 下注册 knowledge-docs 路由
func Register(group *gin.RouterGroup, h *Handler) {
	docs := group.Group("/knowledge-docs")
	docs.GET("", h.List)
	docs.POST("", h.Create)
	docs.GET("/:id", h.Get)
	docs.DELETE("/:id", h.Delete)
	docs.POST("/:id/retry", h.Retry)
}
