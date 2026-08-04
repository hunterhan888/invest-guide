package auth

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Register 在 v1 路由组下注册 auth 公开路由（注册/登录）
func Register(group *gin.RouterGroup, h *Handler) {
	group.POST("/register", h.Register)
	group.POST("/login", h.Login)
}

// AuthenticatorAdapter 把 *Service 包装成 middleware.Authenticator
// 这样 middleware 包不直接依赖 domain/auth
type AuthenticatorAdapter struct {
	Service *Service
}

func (a *AuthenticatorAdapter) Authenticate(ctx context.Context, token string) (string, error) {
	user, err := a.Service.Authenticate(ctx, token)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}
