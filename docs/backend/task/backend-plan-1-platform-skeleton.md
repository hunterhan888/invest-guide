# 后端 Plan 1 — 基础层骨架 Implementation Plan

**Goal:** 搭好后端基础层与启动入口，让 `make dev` 起服后 `GET /api/v1/system/health` 返回 200，并在本 plan 中交付 `domain/system` 的 health/version 端点作为第一个领域模块示范。

**Architecture:** Gin + GORM + PostgreSQL/pgvector；`internal/platform/` 提供基础能力（config/logger/database/response/cache/taskqueue/middleware/router），`internal/domain/` 放领域模块；装配唯一发生在 `cmd/server/main.go`；本 plan 不实现 auth/RAG/LLM，只搭骨架与 system 模块。

**Tech Stack:** Go 1.26、Gin、GORM、pgvector、golang-jwt、golang-migrate、SQLite（测试用）、`log/slog`、testify、rcrowley/go-metrics（限流用 golang.org/x/time/rate）。

**关联设计：** `ARCHITECTURE.md`（后端模块职责 / 依赖规则 / API 约定 / 配置管理 / 安全模型 / 可观测性 / 测试策略）

**前置条件：** 工作目录为仓库根；已安装 Go 1.26+、PostgreSQL 18（含 pgvector 扩展）或使用 docker-compose。

---

## 文件结构

本 plan 创建的文件及其单一职责：

```
backend/
├── go.mod
├── cmd/server/main.go                              # 唯一装配点
├── internal/
│   ├── platform/
│   │   ├── config/
│   │   │   ├── config.go                           # Config 结构体 + Load()
│   │   │   └── config_test.go
│   │   ├── logger/
│   │   │   ├── logger.go                           # slog 初始化 + FromContext
│   │   │   └── logger_test.go
│   │   ├── database/
│   │   │   ├── database.go                         # Connect + Migrate + 事务辅助
│   │   │   └── database_test.go
│   │   ├── response/
│   │   │   ├── response.go                         # Ok/Fail + ErrorResponse + 错误码
│   │   │   └── response_test.go
│   │   ├── cache/
│   │   │   ├── cache.go                            # Cache 接口
│   │   │   ├── lru.go                              # LRU 内存实现
│   │   │   └── lru_test.go
│   │   ├── taskqueue/
│   │   │   ├── taskqueue.go                        # Queue 接口
│   │   │   ├── goroutine_pool.go                  # goroutine pool + channel 实现
│   │   │   └── goroutine_pool_test.go
│   │   ├── middleware/
│   │   │   ├── middleware.go                       # CORS/RequestID/Logger/Recovery
│   │   │   ├── ratelimit.go                        # 限流（按 IP / 用户）
│   │   │   └── middleware_test.go
│   │   └── router/
│   │       ├── router.go                           # New(deps) 装配中间件 + 路由组
│   │       ├── deps.go                             # Deps 容器（依赖注入）
│   │       └── router_test.go
│   └── domain/
│       └── system/
│           ├── route.go                            # Register(group)
│           ├── handler.go                          # health / version handler
│           ├── service.go                          # 业务逻辑（版本信息来源）
│           └── model.go                            # 响应结构体
├── migrations/
│   ├── 0001_init.up.sql                            # pgvector 扩展 + 基础表
│   └── 0001_init.down.sql
└── tests/
    └── e2e/
        └── health_test.go                          # 完整 HTTP 链路验证
```

每个文件只干一件事；`platform/` 各模块互不依赖（除 `config`/`logger` 被多处读取），领域模块只依赖接口。

---

### Task 0: 初始化 go.mod 与项目骨架

**Files:**
- Create: `backend/go.mod`
- Create: `backend/.gitignore`

- [ ] **Step 1: 初始化 module**

Run:
```bash
cd backend && go mod init github.com/invest-guide/backend
```
Expected: 生成 `backend/go.mod`，module path 为 `github.com/invest-guide/backend`，go 版本 `1.26`

- [ ] **Step 2: 写入 `.gitignore`**

`backend/.gitignore`:
```
/bin/
/dist/
*.db
*.out
.env
coverage.txt
```

- [ ] **Step 3: 添加直接依赖**

Run:
```bash
cd backend && go get \
  github.com/gin-gonic/gin@latest \
  gorm.io/gorm@latest \
  gorm.io/driver/postgres@latest \
  gorm.io/driver/sqlite@latest \
  github.com/pgvector/pgvector-go@latest \
  github.com/golang-jwt/jwt/v5@latest \
  golang.org/x/crypto@latest \
  github.com/google/uuid@latest \
  github.com/stretchr/testify@latest \
  golang.org/x/time@latest
```
Expected: `go.mod` 含上述依赖；`go.sum` 生成

- [ ] **Step 4: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无输出（无错误）。`go vet ./...` 也无告警

- [ ] **Step 5: Commit**

```bash
git add backend/go.mod backend/go.sum backend/.gitignore
git commit -m "chore(backend): init go module with core dependencies"
```

---

### Task 1: `config` 模块

**Files:**
- Create: `backend/internal/platform/config/config.go`
- Create: `backend/internal/platform/config/config_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/config/config_test.go`:
```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-32chars-minimum-length!")
	t.Setenv("LLM_API_KEY", "test-key")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "sqlite://dev.db", cfg.DatabaseURL)
	assert.Equal(t, "24h", cfg.JWTExpiry)
	assert.Equal(t, "gpt-4o", cfg.LLMModel)
	assert.Equal(t, "1024", cfg.EmbeddingDim)
	assert.Equal(t, 60, cfg.RateLimitAPI)
}

func TestLoad_MissingRequired_FailsFast(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("LLM_API_KEY", "")
	_, err := Load()
	assert.Error(t, err)
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("JWT_SECRET", "test-secret-32chars-minimum-length!")
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_MODEL", "deepseek-chat")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "deepseek-chat", cfg.LLMModel)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/config/`
Expected: FAIL — `config.Load` 未定义

- [ ] **Step 3: 写最小实现**

`backend/internal/platform/config/config.go`:
```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	JWTExpiry         time.Duration
	LLMBaseURL        string
	LLMAPIKey         string
	LLMModel          string
	LLMTimeout        time.Duration
	LLMStreamTimeout  time.Duration
	EmbeddingBaseURL  string
	EmbeddingAPIKey   string
	EmbeddingModel    string
	EmbeddingDim      string
	CORSOrigins       string
	LogLevel          string
	RateLimitAPI      int
	RateLimitSensitive int
}

func Load() (*Config, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	llmKey := os.Getenv("LLM_API_KEY")
	if llmKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required")
	}

	jwtExpiry, err := time.ParseDuration(envOrDefault("JWT_EXPIRY", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}
	llmTimeout, err := time.ParseDuration(envOrDefault("LLM_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_TIMEOUT: %w", err)
	}
	llmStreamTimeout, err := time.ParseDuration(envOrDefault("LLM_STREAM_TIMEOUT", "120s"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_STREAM_TIMEOUT: %w", err)
	}

	return &Config{
		Port:               envOrDefault("PORT", "8080"),
		DatabaseURL:        envOrDefault("DATABASE_URL", "sqlite://dev.db"),
		JWTSecret:          jwtSecret,
		JWTExpiry:          jwtExpiry,
		LLMBaseURL:         envOrDefault("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMAPIKey:          llmKey,
		LLMModel:           envOrDefault("LLM_MODEL", "gpt-4o"),
		LLMTimeout:         llmTimeout,
		LLMStreamTimeout:   llmStreamTimeout,
		EmbeddingBaseURL:   envOrDefault("EMBEDDING_BASE_URL", envOrDefault("LLM_BASE_URL", "https://api.openai.com/v1")),
		EmbeddingAPIKey:    envOrDefault("EMBEDDING_API_KEY", llmKey),
		EmbeddingModel:     envOrDefault("EMBEDDING_MODEL", "Qwen/Qwen3-Embedding-0.6B"),
		EmbeddingDim:       envOrDefault("EMBEDDING_DIM", "1024"),
		CORSOrigins:        envOrDefault("CORS_ORIGINS", "http://localhost:5173"),
		LogLevel:           envOrDefault("LOG_LEVEL", "info"),
		RateLimitAPI:       envIntOrDefault("RATE_LIMIT_API", 60),
		RateLimitSensitive: envIntOrDefault("RATE_LIMIT_SENSITIVE", 20),
	}, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// HTTPAddr 返回监听地址，便于 main.go 直接调用
func (c *Config) HTTPAddr() string { return ":" + c.Port }
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/config/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/config/
git commit -m "feat(backend/config): add Config + Load with env defaults"
```

---

### Task 2: `logger` 模块

**Files:**
- Create: `backend/internal/platform/logger/logger.go`
- Create: `backend/internal/platform/logger/logger_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/logger/logger_test.go`:
```go
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitJSON(t *testing.T) {
	var buf bytes.Buffer
	Init(&buf, "debug")
	logger.Info("hello", "key", "val")

	var entry map[string]interface{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "hello", entry["msg"])
	assert.Equal(t, "val", entry["key"])
	assert.Equal(t, "INFO", entry["level"])
}

func TestFromContext_WithRequestID(t *testing.T) {
	var buf bytes.Buffer
	Init(&buf, "info")
	ctx := WithRequestID(context.Background(), "req-123")
	FromContext(ctx).Info("scoped")
	var entry map[string]interface{}
	_ = json.Unmarshal(buf.Bytes(), &entry)
	assert.Equal(t, "req-123", entry["request_id"])
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/logger/`
Expected: FAIL — `Init` / `WithRequestID` / `FromContext` 未定义

- [ ] **Step 3: 写最小实现**

`backend/internal/platform/logger/logger.go`:
```go
package logger

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

type ctxKey struct{}

var defaultLogger *slog.Logger

// Init 初始化全局 JSON logger。level: debug/info/warn/error
func Init(w io.Writer, level string) {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	defaultLogger = slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(defaultLogger)
}

// WithRequestID 把 request_id 写入 context
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext 取出可携带 request_id 的 logger；无 ctx 时回退默认 logger
func FromContext(ctx context.Context) *slog.Logger {
	if id, ok := ctx.Value(ctxKey{}).(string); ok && id != "" {
		return defaultLogger.With("request_id", id)
	}
	return defaultLogger
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/logger/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/logger/
git commit -m "feat(backend/logger): add slog JSON logger with request_id context"
```

---

### Task 3: `response` 模块

**Files:**
- Create: `backend/internal/platform/response/response.go`
- Create: `backend/internal/platform/response/response_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/response/response_test.go`:
```go
package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

func TestOk_SuccessNoMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Ok(c, http.StatusOK, gin.H{"id": "1"})

	var body SuccessResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Success)
	assert.Equal(t, "1", body.Data["id"])
	assert.Nil(t, body.Message)
}

func TestFail_InvalidInput(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Fail(c, ErrInvalidInput)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Success)
	assert.Equal(t, "INVALID_INPUT", body.Code)
}

func TestFail_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Fail(c, ErrNotFound)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFail_WrappedErrorStillMatches(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	wrapped := fmt.Errorf("ctx: %w", ErrNotFound)
	// 即使被包装，errors.Is 应识别为 404
	Fail(c, wrapped)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

> 注：第四个测试要求 `Fail` 用 `errors.Is` 识别被 `%w` 包装的 sentinel error，对应 ARCHITECTURE.md 错误传播规则。
> 测试需额外 `import "fmt"`。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/response/`
Expected: FAIL — `Ok` / `Fail` / `SuccessResponse` 等未定义

- [ ] **Step 3: 写最小实现**

`backend/internal/platform/response/response.go`:
```go
package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 领域 sentinel 错误 — 任何 service/repository 返回的错误都通过 errors.Is 匹配这些
var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrRateLimited    = errors.New("rate limited")
	ErrInternal       = errors.New("internal error")
	ErrBadGateway     = errors.New("bad gateway")
	ErrGatewayTimeout = errors.New("gateway timeout")
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code"`
}

func Ok(c *gin.Context, status int, data interface{}) {
	c.JSON(status, SuccessResponse{Success: true, Data: data})
}

func OkWithMessage(c *gin.Context, status int, data interface{}, message string) {
	c.JSON(status, SuccessResponse{Success: true, Data: data, Message: message})
}

func Fail(c *gin.Context, err error) {
	status, code := mapError(err)
	// 5xx 不向客户端泄露内部细节
	msg := err.Error()
	if status >= 500 {
		slog.Error("request failed", "err", err)
		msg = "internal error"
	}
	c.JSON(status, ErrorResponse{Success: false, Error: msg, Code: code})
}

func mapError(err error) (status int, code string) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest, "INVALID_INPUT"
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, ErrConflict):
		return http.StatusConflict, "CONFLICT"
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests, "RATE_LIMITED"
	case errors.Is(err, ErrBadGateway):
		return http.StatusBadGateway, "BAD_GATEWAY"
	case errors.Is(err, ErrGatewayTimeout):
		return http.StatusGatewayTimeout, "GATEWAY_TIMEOUT"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/response/`
Expected: PASS

> 若第四个 wrapped test 失败：把 `errors.New("ctx: " + ErrNotFound.Error())` 改用 `fmt.Errorf("ctx: %w", ErrNotFound)`，因为 `errors.Is` 只对 `%w` 包装链生效。这是测试自身的修正——实现已正确。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/response/
git commit -m "feat(backend/response): add Ok/Fail with error code mapping"
```

---

### Task 4: `database` 模块

**Files:**
- Create: `backend/internal/platform/database/database.go`
- Create: `backend/internal/platform/database/database_test.go`
- Create: `backend/migrations/0001_init.up.sql`
- Create: `backend/migrations/0001_init.down.sql`

> 注：本 task 只创建 migrations 文件框架，schema 仅含 pgvector 扩展声明与一张迁移历史占位；真实业务表在各领域 plan 落地。

- [ ] **Step 1: 写迁移文件**

`backend/migrations/0001_init.up.sql`:
```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

`backend/migrations/0001_init.down.sql`:
```sql
-- 不删除 vector 扩展，避免影响其他依赖
-- 本文件保留以备后续扩展
```

- [ ] **Step 2: 写失败测试**

`backend/internal/platform/database/database_test.go`:
```go
package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestNewSQLite_TemporaryDB(t *testing.T) {
	db, err := NewSQLite(":memory:")
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// 简单建表验证连通
	assert.NoError(t, db.Exec("CREATE TABLE probe (id INTEGER PRIMARY KEY)").Error)
}

func TestNewSQLite_PersistsData(t *testing.T) {
	db, _ := NewSQLite(":memory:")
	db.Exec("CREATE TABLE note (text TEXT)")
	db.Exec("INSERT INTO note (text) VALUES ('hello')")

	var got string
	assert.NoError(t, db.Raw("SELECT text FROM note").Scan(&got).Error)
	assert.Equal(t, "hello", got)
}
```

> 注：测试仅使用 `*gorm.DB` 返回值，无需显式 import `gorm` 包；如 IDE 自动补了 import 但未使用，跑 `gofmt -s` + `goimports` 清理。

- [ ] **Step 3: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/database/`
Expected: FAIL — `NewSQLite` 未定义

- [ ] **Step 4: 写最小实现**

`backend/internal/platform/database/database.go`:
```go
package database

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect 按连接串前缀分发到对应驱动
//   postgres://... 或 postgresql://... → PostgreSQL
//   sqlite://...                       → SQLite（文件或 :memory:）
func Connect(dsn string) (*gorm.DB, error) {
	switch {
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	case strings.HasPrefix(dsn, "sqlite://"):
		return NewSQLite(strings.TrimPrefix(dsn, "sqlite://"))
	default:
		return nil, fmt.Errorf("unsupported database url: %s", dsn)
	}
}

// NewSQLite 创建内存或文件 SQLite，供测试与本地开发使用
func NewSQLite(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
}

// TransactionWrap 在事务中执行 fn，错误自动回滚
func TransactionWrap(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
```

- [ ] **Step 5: 跑测试通过**

Run: `cd backend && go test ./internal/platform/database/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/database/ backend/migrations/
git commit -m "feat(backend/database): add Connect + SQLite helper + migration scaffold"
```

---

### Task 5: `cache` 模块（接口 + LRU 实现）

**Files:**
- Create: `backend/internal/platform/cache/cache.go`
- Create: `backend/internal/platform/cache/lru.go`
- Create: `backend/internal/platform/cache/lru_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/cache/lru_test.go`:
```go
package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLRU_SetGet(t *testing.T) {
	c := NewLRU(2, time.Hour)
	c.Set("a", "1")
	v, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "1", v)
}

func TestLRU_EvictsOnCapacity(t *testing.T) {
	c := NewLRU(2, time.Hour)
	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3") // 应淘汰 a
	_, ok := c.Get("a")
	assert.False(t, ok)
	_, ok = c.Get("b")
	assert.True(t, ok)
}

func TestLRU_ExpiresAfterTTL(t *testing.T) {
	c := NewLRU(2, 10*time.Millisecond)
	c.Set("a", "1")
	time.Sleep(20 * time.Millisecond)
	_, ok := c.Get("a")
	assert.False(t, ok)
}

func TestLRU_Delete(t *testing.T) {
	c := NewLRU(2, time.Hour)
	c.Set("a", "1")
	c.Delete("a")
	_, ok := c.Get("a")
	assert.False(t, ok)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/cache/`
Expected: FAIL — `NewLRU` 未定义

- [ ] **Step 3: 写最小实现**

`backend/internal/platform/cache/cache.go`:
```go
package cache

import "time"

// Cache 缓存抽象；生产可切 Redis 实现，开发用 LRU
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
}
```

`backend/internal/platform/cache/lru.go`:
```go
package cache

import (
	"container/list"
	"sync"
	"time"
)

type lruEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

type LRU struct {
	capacity int
	ttl      time.Duration
	mu       sync.Mutex
	order    *list.List
	items    map[string]*list.Element
}

func NewLRU(capacity int, ttl time.Duration) *LRU {
	return &LRU{
		capacity: capacity,
		ttl:      ttl,
		order:    list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (l *LRU) Get(key string) (interface{}, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	el, ok := l.items[key]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*lruEntry)
	if time.Now().After(entry.expiresAt) {
		l.removeElement(el)
		return nil, false
	}
	l.order.MoveToFront(el)
	return entry.value, true
}

func (l *LRU) Set(key string, value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.items[key]; ok {
		entry := el.Value.(*lruEntry)
		entry.value = value
		entry.expiresAt = time.Now().Add(l.ttl)
		l.order.MoveToFront(el)
		return
	}
	entry := &lruEntry{key: key, value: value, expiresAt: time.Now().Add(l.ttl)}
	el := l.order.PushFront(entry)
	l.items[key] = el
	if l.order.Len() > l.capacity {
		l.evict()
	}
}

func (l *LRU) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.items[key]; ok {
		l.removeElement(el)
	}
}

func (l *LRU) removeElement(el *list.Element) {
	entry := el.Value.(*lruEntry)
	l.order.Remove(el)
	delete(l.items, entry.key)
}

func (l *LRU) evict() {
	el := l.order.Back()
	if el != nil {
		l.removeElement(el)
	}
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/cache/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/cache/
git commit -m "feat(backend/cache): add Cache interface + LRU memory implementation"
```

---

### Task 6: `taskqueue` 模块（接口 + goroutine pool 实现）

**Files:**
- Create: `backend/internal/platform/taskqueue/taskqueue.go`
- Create: `backend/internal/platform/taskqueue/goroutine_pool.go`
- Create: `backend/internal/platform/taskqueue/goroutine_pool_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/taskqueue/goroutine_pool_test.go`:
```go
package taskqueue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGoroutinePool_ExecutesTask(t *testing.T) {
	q := NewGoroutinePool(2, 4)
	defer q.Close(context.Background())

	var count int32
	q.Enqueue(func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	assert.Eventually(t, func() bool { return atomic.LoadInt32(&count) == 1 }, time.Second, 10*time.Millisecond)
}

func TestGoroutinePool_CancelStopsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	q := NewGoroutinePool(2, 4)
	cancel() // 先取消，再关闭应快速返回
	assert.NoError(t, q.Close(ctx))
}

func TestGoroutinePool_ProcessesMultiple(t *testing.T) {
	q := NewGoroutinePool(4, 8)
	defer q.Close(context.Background())

	var count int32
	for i := 0; i < 10; i++ {
		q.Enqueue(func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}
	assert.Eventually(t, func() bool { return atomic.LoadInt32(&count) == 10 }, 2*time.Second, 20*time.Millisecond)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/taskqueue/`
Expected: FAIL — `NewGoroutinePool` / `Queue` 接口未定义

- [ ] **Step 3: 写最小实现**

`backend/internal/platform/taskqueue/taskqueue.go`:
```go
package taskqueue

import "context"

// Task 是异步入库等场景提交的工作单元
type Task func(ctx context.Context) error

// Queue 抽象；生产可切 Redis，开发用内存 goroutine pool
type Queue interface {
	Enqueue(task Task) error
	Close(ctx context.Context) error
}
```

`backend/internal/platform/taskqueue/goroutine_pool.go`:
```go
package taskqueue

import (
	"context"
	"sync"
)

type GoroutinePool struct {
	wg      sync.WaitGroup
	tasks   chan Task
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
	mu      sync.Mutex
}

func NewGoroutinePool(workers, buffer int) *GoroutinePool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &GoroutinePool{
		tasks:  make(chan Task, buffer),
		ctx:    ctx,
		cancel: cancel,
	}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

func (p *GoroutinePool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			_ = task(p.ctx)
		}
	}
}

func (p *GoroutinePool) Enqueue(task Task) error {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return p.ctx.Err()
	}
	p.mu.Unlock()
	select {
	case p.tasks <- task:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

func (p *GoroutinePool) Close(ctx context.Context) error {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return nil
	}
	p.stopped = true
	p.mu.Unlock()

	p.cancel()
	close(p.tasks)
	p.wg.Wait()
	return nil
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/taskqueue/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/taskqueue/
git commit -m "feat(backend/taskqueue): add Queue interface + goroutine pool"
```

---

### Task 7: `middleware` 模块（CORS / RequestID / Logger / Recovery / RateLimit）

**Files:**
- Create: `backend/internal/platform/middleware/middleware.go`
- Create: `backend/internal/platform/middleware/ratelimit.go`
- Create: `backend/internal/platform/middleware/middleware_test.go`

> Auth 中间件留到 Plan 2 与 auth 模块同 plan 落地，本 plan 只留接口与 stub。

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/middleware/middleware_test.go`:
```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRequestID_InjectsHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("request_id"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.Equal(t, w.Header().Get("X-Request-ID"), w.Body.String())
}

func TestCORS_AddsAllowOrigin(t *testing.T) {
	r := gin.New()
	r.Use(CORS("http://localhost:5173"))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	r.ServeHTTP(w, req)
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestRecovery_CapturesPanic(t *testing.T) {
	r := gin.New()
	r.Use(Recovery())
	r.GET("/boom", func(c *gin.Context) { panic("x") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/boom", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	r := gin.New()
	r.Use(RateLimit(2, 1)) // 2 次/秒
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/middleware/`
Expected: FAIL — 各中间件未定义

- [ ] **Step 3: 写最小实现**

`backend/internal/platform/middleware/middleware.go`:
```go
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID 为每个请求生成 UUID，注入 ctx、header、gin.Context，供日志关联
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// CORS 仅允许配置的来源（逗号分隔取第一个匹配）
func CORS(allowedOrigins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if isAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func isAllowed(origin, csv string) bool {
	if origin == "" || csv == "" {
		return false
	}
	for _, o := range splitCSV(csv) {
		if o == origin {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// Recovery 捕获 panic，输出 500，绝不向客户端暴露堆栈
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				_ = debug.Stack()
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "internal error",
					"code":    "INTERNAL_ERROR",
				})
			}
		}()
		c.Next()
	}
}
```

`backend/internal/platform/middleware/ratelimit.go`:
```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimit 按 IP 限流：每秒 rate 次请求，突发 burst
func RateLimit(ratePerSec, burst int) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		buckets  = make(map[string]*rate.Limiter)
		lastSeen = make(map[string]time.Time)
	)
	go cleanupBuckets(buckets, lastSeen, &mu)

	return func(c *gin.Context) {
		key := c.ClientIP()
		mu.Lock()
		l, ok := buckets[key]
		if !ok {
			l = rate.NewLimiter(rate.Limit(ratePerSec), burst)
			buckets[key] = l
		}
		lastSeen[key] = time.Now()
		mu.Unlock()

		if !l.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "rate limited",
				"code":    "RATE_LIMITED",
			})
			return
		}
		c.Next()
	}
}

func cleanupBuckets(buckets map[string]*rate.Limiter, lastSeen map[string]time.Time, mu *sync.Mutex) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		mu.Lock()
		for k, t := range lastSeen {
			if time.Since(t) > 3*time.Minute {
				delete(buckets, k)
				delete(lastSeen, k)
			}
		}
		mu.Unlock()
	}
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/middleware/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/middleware/
git commit -m "feat(backend/middleware): add RequestID/CORS/Recovery/RateLimit"
```

---

### Task 8: `domain/system` 领域模块（health + version）

**Files:**
- Create: `backend/internal/domain/system/route.go`
- Create: `backend/internal/domain/system/handler.go`
- Create: `backend/internal/domain/system/service.go`
- Create: `backend/internal/domain/system/model.go`

> 遵守 ARCHITECTURE.md 领域模块标准结构（route/handler/service/model，本模块无 repository 因为不访问 DB）。`models` 端点（GET /api/v1/system/models）需要 LLM 配置，留到 Plan 4 本 plan 只做 health/version。

- [ ] **Step 1: 写 model.go**

`backend/internal/domain/system/model.go`:
```go
package system

type HealthResponse struct {
	Status string `json:"status"`
}

type VersionResponse struct {
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
}
```

- [ ] **Step 2: 写 service.go**

`backend/internal/domain/system/service.go`:
```go
package system

import "runtime"

type Service struct {
	version string
}

func NewService(version string) *Service {
	return &Service{version: version}
}

func (s *Service) Health() HealthResponse {
	return HealthResponse{Status: "ok"}
}

func (s *Service) Version() VersionResponse {
	return VersionResponse{Version: s.version, GoVersion: runtime.Version()}
}
```

- [ ] **Step 3: 写 handler.go**

`backend/internal/domain/system/handler.go`:
```go
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
```

- [ ] **Step 4: 写 route.go**

`backend/internal/domain/system/route.go`:
```go
package system

import "github.com/gin-gonic/gin"

// Register 在公共路由组下注册 system 端点（无鉴权）
func Register(group *gin.RouterGroup, h *Handler) {
	group.GET("/health", h.Health)
	group.GET("/version", h.Version)
}
```

- [ ] **Step 5: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 6: Commit**

```bash
git add backend/internal/domain/system/
git commit -m "feat(backend/system): add health and version endpoints"
```

---

### Task 9: `router` 模块 + 装配 `Deps`

**Files:**
- Create: `backend/internal/platform/router/deps.go`
- Create: `backend/internal/platform/router/router.go`
- Create: `backend/internal/platform/router/router_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/router/router_test.go`:
```go
package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint_Returns200(t *testing.T) {
	deps := NewTestDeps()
	r := New(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
	assert.Equal(t, "ok", body["data"].(map[string]interface{})["status"])
}

func TestVersionEndpoint_ReturnsVersion(t *testing.T) {
	deps := NewTestDeps()
	deps.Version = "0.0.1-test"
	r := New(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/system/version", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "0.0.1-test", data["version"])
}

func TestUnknownRoute_Returns404(t *testing.T) {
	deps := NewTestDeps()
	r := New(deps)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/nope", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/router/`
Expected: FAIL — `Deps` / `New` / `NewTestDeps` 未定义

- [ ] **Step 3: 写 deps.go**

`backend/internal/platform/router/deps.go`:
```go
package router

import (
	"github.com/invest-guide/backend/internal/domain/system"
	"github.com/invest-guide/backend/internal/platform/cache"
	"github.com/invest-guide/backend/internal/platform/config"
	"github.com/invest-guide/backend/internal/platform/taskqueue"
	"gorm.io/gorm"
)

// Deps 是装配中心构造的依赖容器，所有 handler 从此处获取依赖
type Deps struct {
	Cfg       *config.Config
	DB        *gorm.DB
	Cache     cache.Cache
	TaskQueue taskqueue.Queue
	Version   string

	SystemHandler *system.Handler
}

// NewTestDeps 仅用于路由测试，构造无 DB 的最小依赖集
func NewTestDeps() *Deps {
	return &Deps{
		Cfg:     &config.Config{CORSOrigins: "*"},
		Version: "0.0.1-test",
		SystemHandler: system.NewHandler(
			system.NewService("0.0.1-test"),
		),
	}
}
```

- [ ] **Step 4: 写 router.go**

`backend/internal/platform/router/router.go`:
```go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/domain/system"
	"github.com/invest-guide/backend/internal/platform/middleware"
)

// New 装配中间件栈与路由组，返回 *gin.Engine
// 中间件顺序按 ARCHITECTURE.md 安全模型：CORS → RequestID → Logger → Recovery → RateLimit
func New(deps *Deps) *gin.Engine {
	r := gin.New()
	r.Use(
		middleware.CORS(deps.Cfg.CORSOrigins),
		middleware.RequestID(),
		middleware.Recovery(),
	)
	if deps.Cfg.RateLimitAPI > 0 {
		r.Use(middleware.RateLimit(deps.Cfg.RateLimitAPI, deps.Cfg.RateLimitAPI))
	}

	v1 := r.Group("/api/v1")
	system.Register(v1.Group("/system"), deps.SystemHandler)
	return r
}
```

- [ ] **Step 5: 跑测试通过**

Run: `cd backend && go test ./internal/platform/router/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/router/
git commit -m "feat(backend/router): wire middleware stack + system routes"
```

---

### Task 10: `cmd/server/main.go` 装配

**Files:**
- Create: `backend/cmd/server/main.go`

- [ ] **Step 1: 写 main.go**

`backend/cmd/server/main.go`:
```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/invest-guide/backend/internal/domain/system"
	"github.com/invest-guide/backend/internal/platform/cache"
	"github.com/invest-guide/backend/internal/platform/config"
	"github.com/invest-guide/backend/internal/platform/database"
	"github.com/invest-guide/backend/internal/platform/logger"
	"github.com/invest-guide/backend/internal/platform/router"
	"github.com/invest-guide/backend/internal/platform/taskqueue"
)

const version = "0.0.1-dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	logger.Init(os.Stdout, cfg.LogLevel)
	slog.Info("starting invest-guide backend", "version", version, "port", cfg.Port)

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect db failed", "err", err)
		os.Exit(1)
	}

	cacheInst := cache.NewLRU(1000, time.Hour)
	taskQ := taskqueue.NewGoroutinePool(4, 16)

	deps := &router.Deps{
		Cfg:           cfg,
		DB:            db,
		Cache:         cacheInst,
		TaskQueue:     taskQ,
		Version:       version,
		SystemHandler: system.NewHandler(system.NewService(version)),
	}

	r := router.New(deps)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "err", err)
	}
	if err := taskQ.Close(shutdownCtx); err != nil {
		slog.Error("taskqueue close error", "err", err)
	}
	slog.Info("stopped")
}
```

- [ ] **Step 2: 验证编译**

Run: `cd backend && go build ./cmd/server/`
Expected: 无错误

- [ ] **Step 3: 验证启动（需要环境变量）**

Run:
```bash
cd backend && \
  JWT_SECRET="dev-secret-32chars-minimum-length!!!" \
  LLM_API_KEY="dev-key" \
  go run ./cmd/server/
```
Expected: 日志输出 "starting invest-guide backend"，进程不退出
按 Ctrl+C → 输出 "shutting down..."、"stopped"，进程退出

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/
git commit -m "feat(backend): wire main entry with graceful shutdown"
```

---

### Task 11: Makefile 入口

**Files:**
- Create: `Makefile`

- [ ] **Step 1: 写 Makefile**

```makefile
.PHONY: dev backend-dev backend-test backend-build frontend-dev frontend-build test

# 后端
backend-dev:
	cd backend && JWT_SECRET="dev-secret-32chars-minimum-length!!!" LLM_API_KEY="dev-key" go run ./cmd/server/

backend-test:
	cd backend && go test ./... -cover

backend-build:
	cd backend && go build -o bin/server ./cmd/server/

backend-vet:
	cd backend && go vet ./...

backend-fmt:
	cd backend && gofmt -l .

# 前端
frontend-dev:
	cd frontend && bun run dev

frontend-build:
	cd frontend && bun run build

# 综合
test: backend-test
	@echo "All tests passed"

dev: backend-dev
```

- [ ] **Step 2: 验证 make 可用**

Run: `make backend-test`
Expected: 全部单元测试通过

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add Makefile with backend dev/test/build targets"
```

---

### Task 12: .env.example 与 docker-compose

**Files:**
- Create: `.env.example`
- Create: `docker-compose.yml`

- [ ] **Step 1: 写 `.env.example`**

```
# 后端必填
JWT_SECRET=change-me-to-a-long-random-string-32-chars-min!
LLM_API_KEY=sk-...

# 后端可选（带默认值）
PORT=8080
DATABASE_URL=postgres://invest:invest@localhost:5432/investguide?sslmode=disable
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=gpt-4o
LLM_TIMEOUT=30s
LLM_STREAM_TIMEOUT=120s
EMBEDDING_MODEL=Qwen/Qwen3-Embedding-0.6B
EMBEDDING_DIM=1024
CORS_ORIGINS=http://localhost:5173
LOG_LEVEL=debug
RATE_LIMIT_API=60
RATE_LIMIT_SENSITIVE=20
```

- [ ] **Step 2: 写 `docker-compose.yml`**

```yaml
services:
  postgres:
    image: pgvector/pgvector:pg18
    environment:
      POSTGRES_USER: invest
      POSTGRES_PASSWORD: invest
      POSTGRES_DB: investguide
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U invest -d investguide"]
      interval: 5s
      timeout: 3s
      retries: 10

volumes:
  pgdata:
```

- [ ] **Step 3: 验证 docker-compose 语法**

Run: `docker compose config > /dev/null`
Expected: 无错误输出（若本机无 docker，跳过本步）

- [ ] **Step 4: Commit**

```bash
git add .env.example docker-compose.yml
git commit -m "build: add .env.example and docker-compose for postgres+pgvector"
```

---

### Task 13: E2E 测试（完整 HTTP 链路）

**Files:**
- Create: `backend/tests/e2e/health_test.go`
- Create: `backend/tests/e2e/helpers.go`

- [ ] **Step 1: 写 helpers.go（共享 E2E 测试夹具）**

`backend/tests/e2e/helpers.go`:
```go
package e2e

import (
	"net/http/httptest"

	"github.com/invest-guide/backend/internal/platform/router"
)

// NewTestServer 构造一个内存中完整装配的 router，供 E2E 测试发起真实 HTTP 请求
func NewTestServer() *httptest.Server {
	deps := router.NewTestDeps()
	r := router.New(deps)
	return httptest.NewServer(r)
}
```

- [ ] **Step 2: 写失败测试**

`backend/tests/e2e/health_test.go`:
```go
package e2e

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthE2E(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/system/health")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, `"status"`)
	assert.Contains(t, bodyStr, `"ok"`)
	assert.Contains(t, bodyStr, `"success"`)
}

func TestVersionE2E(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/system/version")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"goVersion"`)
}

func TestRequestIDHeader_Present(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/system/health")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

func TestUnknownRoute_Returns404(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/no-such-route")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
```

- [ ] **Step 3: 跑测试通过**

Run: `cd backend && go test ./tests/e2e/`
Expected: PASS（4 个测试全过）

- [ ] **Step 4: Commit**

```bash
git add backend/tests/e2e/
git commit -m "test(backend/e2e): add system health/version end-to-end tests"
```

---

### Task 14: 最终验证与覆盖率

**Files:** 无修改

- [ ] **Step 1: 全量格式化**

Run: `cd backend && gofmt -l .`
Expected: 无输出（无文件需格式化）

- [ ] **Step 2: 全量 vet**

Run: `cd backend && go vet ./...`
Expected: 无告警

- [ ] **Step 3: 全量测试 + 覆盖率**

Run: `cd backend && go test ./... -cover`
Expected: 全部 PASS；`internal/` 覆盖率 ≥ 70%（Plan 1 因无业务逻辑，主要在 config/middleware/cache/taskqueue/response/router，应在 70%+）

- [ ] **Step 4: 启动验证**

启动 docker-compose（或本地 Postgres），按下述运行：
```bash
make backend-dev
```
另开终端：
```bash
curl -i http://localhost:8080/api/v1/system/health
curl -i http://localhost:8080/api/v1/system/version
curl -i http://localhost:8080/api/v1/nope
```
Expected:
- `/health` 返回 `200` + `{"success":true,"data":{"status":"ok"}}`
- `/version` 返回 `200` + `{"success":true,"data":{"version":"0.0.1-dev","goVersion":"go1.26..."}}`
- `/nope` 返回 `404`
- 三个响应都含 `X-Request-ID` header
停止 backend-dev

- [ ] **Step 5: 文档自洽性核对**

Run: `grep -n "internal/<module>\|backend/internal/" AGENT.md ARCHITECTURE.md`
Expected: 路径与实际结构一致（`internal/platform/` 与 `internal/domain/`）

- [ ] **Step 6: 总结提交（若有零散改动）**

```bash
git status
```
若有未提交改动 → 按需提交；否则无操作。

- [ ] **Step 7: 标记完成**

Plan 1 至此完成。后端基础层骨架可启动、可响应 health/version、有完整测试覆盖、有 graceful shutdown。下一个 plan（Plan 2: auth 领域）可基于此骨架落地。
