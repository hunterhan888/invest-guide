package router

import (
	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/domain/auth"
	"github.com/invest-guide/backend/internal/domain/system"
	"github.com/invest-guide/backend/internal/platform/middleware"
)

// New 装配中间件栈与路由组，返回 *gin.Engine
// 中间件顺序按 ARCHITECTURE.md 安全模型：CORS → RequestID → Recovery → RateLimit
func New(deps *Deps) *gin.Engine {
	r := gin.New()
	r.Use(
		middleware.CORS(deps.Cfg.CORSOrigins),
		middleware.RequestID(),
		middleware.RequestLogger(),
		middleware.Recovery(),
	)
	if deps.Cfg.RateLimitAPI > 0 {
		r.Use(middleware.RateLimit(deps.Cfg.RateLimitAPI, deps.Cfg.RateLimitAPI))
	}

	v1 := r.Group("/api/v1")
	system.Register(v1.Group("/system"), deps.SystemHandler, deps.ModelsHandler)

	// auth 公开路由 + 独立失败限流（5 次失败/15 分钟，按 IP）
	if deps.AuthHandler != nil {
		authPublic := v1.Group("/auth")
		if deps.LoginRateLimit != nil {
			authPublic.Use(deps.LoginRateLimit.Handler())
		}
		auth.Register(authPublic, deps.AuthHandler)
	}

	// 鉴权路由组（后续 plan 通过 PrivateRoutes 挂 conversations/knowledge-docs）
	if deps.Authenticator != nil {
		v1Auth := v1.Group("")
		v1Auth.Use(middleware.Auth(deps.Authenticator))
		deps.RegisterPrivateRoutes(v1Auth)
	}
	return r
}
