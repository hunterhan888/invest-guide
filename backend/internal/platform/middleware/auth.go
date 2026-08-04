package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/platform/response"
)

// Authenticator 是 auth.Service 满足的最小接口（避免 middleware 反向依赖 domain/auth）
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (userID string, err error)
}

func Auth(a Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			response.Fail(c, response.ErrUnauthorized)
			c.Abort()
			return
		}
		userID, err := a.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.Fail(c, response.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Set("userID", userID)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) string {
	if v, ok := c.Get("userID"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimPrefix(h, prefix)
}
