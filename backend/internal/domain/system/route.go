package system

import "github.com/gin-gonic/gin"

// Register 在公共路由组下注册 system 端点（无鉴权）
func Register(group *gin.RouterGroup, h *Handler, mh *ModelsHandler) {
	group.GET("/health", h.Health)
	group.GET("/version", h.Version)
	group.GET("/models", mh.Models)
}
