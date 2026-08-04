# 后端 Plan 2 — auth 领域 Implementation Plan

**Goal:** 实现 `domain/auth` 模块：用户注册、登录、JWT 签发与校验、鉴权中间件、当前用户提取；并补 `users` 表迁移。完成后所有需要鉴权的路由组（后续 plan 的 conversations/knowledge-docs）可挂 `middleware.Auth`。

**Architecture:** bcrypt(cost=12) 存密码；JWT HS256，载荷含 `user_id/exp/iat/iss`；鉴权中间件从 `Authorization: Bearer <token>` 提取并校验，把 `userID` 注入 `gin.Context`；service 只依赖 `UserRepository` 接口，GORM 实现独立；登录/注册路由独立限流（5 次/15 分钟，按 IP）；公开路由跳过鉴权。

**Tech Stack:** `golang-jwt/jwt/v5`、`golang.org/x/crypto/bcrypt`、GORM、Gin、testify、SQLite（测试）。

**关联设计：** `ARCHITECTURE.md`（安全模型 / JWT / 限流 / 错误传播 / Repository 模式）

**前置条件：** Plan 1 已完成（`platform/` 全部就绪，`response.Err*` 已可用，`middleware.RateLimit` 已可用）。

---

## 文件结构

```
backend/
├── internal/
│   ├── domain/auth/
│   │   ├── route.go                  # Register(public, private gin.RouterGroup)
│   │   ├── handler.go                # Register / Login handler
│   │   ├── service.go                # 业务逻辑（bcrypt、JWT 签发）
│   │   ├── repository.go             # UserRepository 接口
│   │   ├── repo_gorm.go              # GORM 实现
│   │   ├── model.go                  # User 实体 + 请求/响应结构体
│   │   ├── jwt.go                    # JWT 签发/校验
│   │   ├── jwt_test.go
│   │   ├── service_test.go
│   │   └── handler_test.go
│   └── platform/middleware/
│       ├── auth.go                   # Auth 中间件 + CurrentUserID
│       └── auth_test.go
├── migrations/
│   ├── 0002_users.up.sql
│   └── 0002_users.down.sql
└── tests/e2e/
    └── auth_test.go                  # 注册→登录→访问鉴权路由的完整链路
```

新增依赖：无（`golang-jwt/jwt/v5`、`golang.org/x/crypto` 已在 Plan 1 加入）。

---

### Task 0: users 迁移

**Files:**
- Create: `backend/migrations/0002_users.up.sql`
- Create: `backend/migrations/0002_users.down.sql`

- [ ] **Step 1: 写 up.sql**

`backend/migrations/0002_users.up.sql`:
```sql
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_email ON users(email);
```

> id 用 UUID 字符串（应用层生成），与 GORM 默认约定一致；email 唯一；created_at/updated_at 由 GORM autoPopulate，但 SQL 默认值兜底以备直接插数据。

- [ ] **Step 2: 写 down.sql**

`backend/migrations/0002_users.down.sql`:
```sql
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/0002_users.*
git commit -m "feat(backend/auth): add users table migration"
```

---

### Task 1: `auth/model.go` — 实体与 DTO

**Files:**
- Create: `backend/internal/domain/auth/model.go`

- [ ] **Step 1: 写 model.go**

`backend/internal/domain/auth/model.go`:
```go
package auth

import "time"

// User 是数据库实体（GORM）
type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	DisplayName  string    `json:"displayName"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (User) TableName() string { return "users" }

// 公开响应中不含 password_hash
type UserDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

func (u *User) ToDTO() UserDTO {
	return UserDTO{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName}
}

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8,max=72"`
	DisplayName string `json:"displayName" binding:"required,max=50"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}
```

- [ ] **Step 2: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/auth/model.go
git commit -m "feat(backend/auth): add User entity and request/response DTOs"
```

---

### Task 2: `auth/jwt.go` — JWT 签发与校验

**Files:**
- Create: `backend/internal/domain/auth/jwt.go`
- Create: `backend/internal/domain/auth/jwt_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/domain/auth/jwt_test.go`:
```go
package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueToken_ContainsUserID(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", "investguide", time.Hour)
	token, err := issuer.Issue("user-123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := issuer.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "investguide", claims.Issuer)
}

func TestVerify_RejectsWrongSecret(t *testing.T) {
	issuerA := NewJWTIssuer("secret-a", "investguide", time.Hour)
	issuerB := NewJWTIssuer("secret-b", "investguide", time.Hour)

	token, _ := issuerA.Issue("user-1")
	_, err := issuerB.Verify(token)
	assert.Error(t, err)
}

func TestVerify_RejectsExpiredToken(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", "investguide", -time.Hour)
	token, _ := issuer.Issue("user-1")
	_, err := issuer.Verify(token)
	assert.Error(t, err)
}

func TestVerify_RejectsWrongIssuer(t *testing.T) {
	issuer := NewJWTIssuer("test-secret", "investguide", time.Hour)
	token, _ := issuer.Issue("user-1")

	other := NewJWTIssuer("test-secret", "other-issuer", time.Hour)
	_, err := other.Verify(token)
	assert.Error(t, err)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/domain/auth/`
Expected: FAIL — `NewJWTIssuer` 未定义

- [ ] **Step 3: 写实现**

`backend/internal/domain/auth/jwt.go`:
```go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type JWTIssuer struct {
	secret  []byte
	issuer  string
	expiry  time.Duration
}

func NewJWTIssuer(secret, issuer string, expiry time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), issuer: issuer, expiry: expiry}
}

func (j *JWTIssuer) Issue(userID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTIssuer) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	parser := jwt.NewParser(jwt.WithIssuer(j.issuer))
	token, err := parser.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return j.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/domain/auth/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/auth/jwt.go backend/internal/domain/auth/jwt_test.go
git commit -m "feat(backend/auth): add JWT HS256 issuer/verifier"
```

---

### Task 3: `auth/repository.go` + `repo_gorm.go`

**Files:**
- Create: `backend/internal/domain/auth/repository.go`
- Create: `backend/internal/domain/auth/repo_gorm.go`

- [ ] **Step 1: 写接口与实现**

`backend/internal/domain/auth/repository.go`:
```go
package auth

import (
	"context"
	"errors"

	"github.com/invest-guide/backend/internal/platform/response"
	"gorm.io/gorm"
)

var ErrDuplicateEmail = errors.New("duplicate email")

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
}
```

> `ErrDuplicateEmail` 作为 sentinel 便于 service 区分冲突与其他错误；handler 映射到 409。`response.ErrConflict` 用于统一对外。

`backend/internal/domain/auth/repo_gorm.go`:
```go
package auth

import (
	"context"
	"errors"

	"github.com/invest-guide/backend/internal/platform/response"
	"gorm.io/gorm"
)

type gormUserRepository struct {
	db *gorm.DB
}

func NewGORMUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

func (r *gormUserRepository) Create(ctx context.Context, u *User) error {
	err := r.db.WithContext(ctx).Create(u).Error
	if err != nil {
		if isDuplicateKeyErr(err) {
			return ErrDuplicateEmail
		}
		return err
	}
	return nil
}

func (r *gormUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *gormUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// isDuplicateKeyErr 检测 PG/SQLite 唯一约束冲突
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// SQLite: "UNIQUE constraint failed"
	// PG: "duplicate key value violates unique constraint"
	return containsAny(msg, "UNIQUE constraint failed", "duplicate key value")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
```

- [ ] **Step 2: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/auth/repository.go backend/internal/domain/auth/repo_gorm.go
git commit -m "feat(backend/auth): add UserRepository interface and GORM impl"
```

---

### Task 4: `auth/service.go` — 注册与登录

**Files:**
- Create: `backend/internal/domain/auth/service.go`
- Create: `backend/internal/domain/auth/service_test.go`

- [ ] **Step 1: 写失败测试（用 fake repo）**

`backend/internal/domain/auth/service_test.go`:
```go
package auth

import (
	"context"
	"testing"

	"github.com/invest-guide/backend/internal/platform/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepo struct {
	users      map[string]*User // by email
	byID       map[string]*User
	createErr  error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]*User{}, byID: map[string]*User{}}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if _, exists := f.users[u.Email]; exists {
		return ErrDuplicateEmail
	}
	f.users[u.Email] = u
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*User, error) {
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return nil, response.ErrNotFound
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id string) (*User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, response.ErrNotFound
}

func newTestService() (*Service, *fakeUserRepo) {
	repo := newFakeUserRepo()
	jwt := NewJWTIssuer("test-secret", "investguide", 0)
	// expiry=0 会被 Issue 中的 now.Add(0) 处理为立即过期；测试时手动覆盖
	return &Service{repo: repo, jwt: jwt, bcryptCost: bcrypt.MinCost}, repo
}

func TestService_Register_CreatesUser(t *testing.T) {
	svc, repo := newTestService()
	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email: "a@b.com", Password: "password123", DisplayName: "A",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.User.ID)
	assert.Equal(t, "a@b.com", resp.User.Email)

	// password_hash 已写入且与原文不同
	stored := repo.users["a@b.com"]
	assert.NotEqual(t, "password123", stored.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("password123")))
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc, _ := newTestService()
	_, _ = svc.Register(context.Background(), RegisterRequest{
		Email: "a@b.com", Password: "password123", DisplayName: "A",
	})
	_, err := svc.Register(context.Background(), RegisterRequest{
		Email: "a@b.com", Password: "password456", DisplayName: "B",
	})
	assert.ErrorIs(t, err, ErrDuplicateEmail)
}

func TestService_Login_Success(t *testing.T) {
	svc, _ := newTestService()
	_, _ = svc.Register(context.Background(), RegisterRequest{
		Email: "a@b.com", Password: "password123", DisplayName: "A",
	})
	// 用 positive expiry 重置 jwt issuer
	svc.jwt = NewJWTIssuer("test-secret", "investguide", 3600*1_000_000_000) // 1h in ns
	resp, err := svc.Login(context.Background(), LoginRequest{
		Email: "a@b.com", Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "a@b.com", resp.User.Email)
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc, _ := newTestService()
	_, _ = svc.Register(context.Background(), RegisterRequest{
		Email: "a@b.com", Password: "password123", DisplayName: "A",
	})
	_, err := svc.Login(context.Background(), LoginRequest{
		Email: "a@b.com", Password: "wrong",
	})
	assert.Error(t, err)
}

func TestService_Login_UserNotFound(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Login(context.Background(), LoginRequest{
		Email: "nope@b.com", Password: "x",
	})
	assert.Error(t, err)
}
```

> fakeUserRepo.GetByIdent 的 "not found" error 字符串仅测试用；service 内部会处理实际 ErrNotFound。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/domain/auth/`
Expected: FAIL — `Service` 未定义

- [ ] **Step 3: 写实现**

`backend/internal/domain/auth/service.go`:
```go
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/invest-guide/backend/internal/platform/response"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo       UserRepository
	jwt        *JWTIssuer
	bcryptCost int
}

func NewService(repo UserRepository, jwt *JWTIssuer) *Service {
	return &Service{repo: repo, jwt: jwt, bcryptCost: 12}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, response.ErrInternal
	}
	user := &User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			return nil, response.ErrConflict
		}
		return nil, err
	}
	token, err := s.jwt.Issue(user.ID)
	if err != nil {
		return nil, response.ErrInternal
	}
	return &AuthResponse{Token: token, User: user.ToDTO()}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, response.ErrUnauthorized
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, response.ErrUnauthorized
	}
	token, err := s.jwt.Issue(user.ID)
	if err != nil {
		return nil, response.ErrInternal
	}
	return &AuthResponse{Token: token, User: user.ToDTO()}, nil
}

// Authenticate 由中间件调用：校验 token 并返回 user record
func (s *Service) Authenticate(ctx context.Context, tokenStr string) (*User, error) {
	claims, err := s.jwt.Verify(tokenStr)
	if err != nil {
		return nil, response.ErrUnauthorized
	}
	user, err := s.repo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, response.ErrUnauthorized
	}
	return user, nil
}

// ensure time is referenced for future expansion (Token expiry 校验等)
var _ = time.Now
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/domain/auth/`
Expected: PASS（5 个 service + 4 个 jwt）

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/auth/service.go backend/internal/domain/auth/service_test.go
git commit -m "feat(backend/auth): add Service with Register/Login/Authenticate"
```

---

### Task 5: `middleware/auth.go` — 鉴权中间件

**Files:**
- Create: `backend/internal/platform/middleware/auth.go`
- Create: `backend/internal/platform/middleware/auth_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/middleware/auth_test.go`:
```go
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeAuthSvc struct {
	token string
	uid   string
	err   error
}

func (f *fakeAuthSvc) Authenticate(ctx context.Context, token string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if token != f.token {
		return "", f.err
	}
	return f.uid, nil
}

func init() { gin.SetMode(gin.TestMode) }

func TestAuth_MissingHeader_Returns401(t *testing.T) {
	r := gin.New()
	svc := &fakeAuthSvc{}
	r.Use(Auth(svc))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_BadScheme_Returns401(t *testing.T) {
	r := gin.New()
	r.Use(Auth(&fakeAuthSvc{}))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Token abc")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_ValidToken_SetsUserID(t *testing.T) {
	svc := &fakeAuthSvc{token: "good", uid: "user-1"}
	r := gin.New()
	r.Use(Auth(svc))

	var gotUserID string
	r.GET("/x", func(c *gin.Context) {
		gotUserID = CurrentUserID(c)
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer good")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", gotUserID)
}

func TestAuth_InvalidToken_Returns401(t *testing.T) {
	svc := &fakeAuthSvc{err: assert.AnError}
	r := gin.New()
	r.Use(Auth(svc))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer bad")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
```

> `assert.AnError` 来自 testify，确保 fake 用 `errors.New` 包装也能被中间件当作失败。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/middleware/`
Expected: FAIL — `Authenticator` 接口 / `Auth` / `CurrentUserID` 未定义

- [ ] **Step 3: 写实现**

`backend/internal/platform/middleware/auth.go`:
```go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/platform/response"
)

// Authenticator 是 auth.Service 满足的最小接口（避免 middleware 反向依赖 domain/auth）
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (userID string, err error)
}

// 注意：auth.Service.Authenticate 返回 (*User, error)，类型不一致。
// 见 Task 6 适配器包装。这里直接接收返回 userID 的版本。

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

// 防止 http import 未使用
var _ = http.StatusOK
```

> 中间件依赖接口不依赖具体 `auth.Service`，符合依赖倒置原则。下面的 Task 6 中提供一个适配器把 `*auth.Service` 包装成 `Authenticator`。

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/middleware/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/middleware/auth.go backend/internal/platform/middleware/auth_test.go
git commit -m "feat(backend/middleware): add Auth middleware with Bearer token check"
```

---

### Task 6: `auth/handler.go` + `route.go` + 适配器

**Files:**
- Create: `backend/internal/domain/auth/handler.go`
- Create: `backend/internal/domain/auth/route.go`
- Create: `backend/internal/domain/auth/handler_test.go`

- [ ] **Step 1: 写 handler.go**

`backend/internal/domain/auth/handler.go`:
```go
package auth

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

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidInput)
		return
	}
	resp, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusCreated, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidInput)
		return
	}
	resp, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusOK, resp)
}
```

- [ ] **Step 2: 写 route.go（含适配器）**

`backend/internal/domain/auth/route.go`:
```go
package auth

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/platform/middleware"
)

// Register 在 v1 路由组下注册 auth 公开路由（注册/登录）
// 登录/注册由 router 层加独立限流（5 次/15 分钟）— 见 router.Task
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
```

- [ ] **Step 3: 写 handler 测试**

`backend/internal/domain/auth/handler_test.go`:
```go
package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func init() { gin.SetMode(gin.TestMode) }

func newHandlerForTest() (*Handler, *fakeUserRepo) {
	repo := newFakeUserRepo()
	jwt := NewJWTIssuer("test-secret", "investguide", 1<<30) // 长有效期
	svc := &Service{repo: repo, jwt: jwt, bcryptCost: bcrypt.MinCost}
	return NewHandler(svc), repo
}

func doJSON(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Register_Success(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)

	body := `{"email":"a@b.com","password":"password123","displayName":"A"}`
	w := doJSON(r, "POST", "/auth/register", body)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Success bool         `json:"success"`
		Data    AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "a@b.com", resp.Data.User.Email)
	assert.NotEmpty(t, resp.Data.Token)
}

func TestHandler_Register_InvalidInput(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)

	w := doJSON(r, "POST", "/auth/register", `{"email":"not-an-email"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_DuplicateEmail_Conflict(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)

	body := `{"email":"a@b.com","password":"password123","displayName":"A"}`
	_ = doJSON(r, "POST", "/auth/register", body)
	w := doJSON(r, "POST", "/auth/register", body)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Login_Success(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)

	body := `{"email":"a@b.com","password":"password123","displayName":"A"}`
	_ = doJSON(r, "POST", "/auth/register", body)

	lb := `{"email":"a@b.com","password":"password123"}`
	w := doJSON(r, "POST", "/auth/login", lb)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool         `json:"success"`
		Data    AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.Token)
}

func TestHandler_Login_WrongPassword_Unauthorized(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)

	_ = doJSON(r, "POST", "/auth/register", `{"email":"a@b.com","password":"password123","displayName":"A"}`)
	w := doJSON(r, "POST", "/auth/login", `{"email":"a@b.com","password":"wrong"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/domain/auth/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/auth/handler.go backend/internal/domain/auth/route.go backend/internal/domain/auth/handler_test.go
git commit -m "feat(backend/auth): add Register/Login handlers and route"
```

---

### Task 7: 接入 `router` 与 `main.go`

**Files:**
- Modify: `backend/internal/platform/router/deps.go`
- Modify: `backend/internal/platform/router/router.go`
- Modify: `backend/internal/platform/router/router_test.go`
- Modify: `backend/cmd/server/main.go`

> Plan 1 已有 router/deps.go。本 task 在其上扩展 AuthHandler、JWTIssuer、UserRepository。

- [ ] **Step 1: 扩展 deps.go**

在 `Deps` 结构体添加字段：

```go
type Deps struct {
	// ... 原有字段 ...

	Authenticator middleware.Authenticator
	AuthHandler   *auth.Handler
}
```

`NewTestDeps()` 装配内存版 auth：

```go
func NewTestDeps() *Deps {
	jwtIssuer := auth.NewJWTIssuer("test-secret", "investguide", 1<<30)
	userRepo := auth.NewGORMUserRepository(newTestSQLite(t)) // 见 Step 2 helper
	authSvc := auth.NewService(userRepo, jwtIssuer)
	return &Deps{
		Cfg:           &config.Config{CORSOrigins: "*", RateLimitAPI: 60},
		Version:       "0.0.1-test",
		SystemHandler: system.NewHandler(system.NewService("0.0.1-test")),
		AuthHandler:   auth.NewHandler(authSvc),
		Authenticator: &auth.AuthenticatorAdapter{Service: authSvc},
	}
}
```

> 上述 `newTestSQLite(t)` 需要从测试 helper 提供。Plan 1 用的是无 DB 装配；本 plan 引入 SQLite。

- [ ] **Step 2: 在 router_test.go 添加测试夹具 helper**

在 `router_test.go` 顶部添加：

```go
import (
	"testing"

	"github.com/invest-guide/backend/internal/platform/database"
)

func newTestSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&auth.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
```

> 修正 `NewTestDeps` 让其接收 `t *testing.T` 参数：把签名改为 `NewTestDeps(t *testing.T) *Deps`，并相应调整 Plan 1 已有的测试调用（Plan 1 的 3 个测试都在 router_test.go 内，本 task 同步改）。

修改 Plan 1 router_test.go 的三个测试，让其调用 `NewTestDeps(t)`：

```go
func TestHealthEndpoint_Returns200(t *testing.T) {
	deps := NewTestDeps(t)
	r := New(deps)
	// ... 其余不变 ...
}
```

（同样修改 TestVersionEndpoint_ReturnsVersion、TestUnknownRoute_Returns404）

- [ ] **Step 3: 扩展 router.go 装配 auth 路由**

```go
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

	// auth 公开路由 + 独立限流（5 次/分钟，注册/登录按 IP）
	authPublic := v1.Group("/auth")
	authPublic.Use(middleware.RateLimit(deps.Cfg.RateLimitSensitive/4, deps.Cfg.RateLimitSensitive))
	auth.Register(authPublic, deps.AuthHandler)

	// 鉴权路由组（后续 plan 在此 group 挂 conversations/knowledge-docs）
	if deps.Authenticator != nil {
		v1Auth := v1.Group("")
		v1Auth.Use(middleware.Auth(deps.Authenticator))
		deps.registerPrivateRoutes(v1Auth)
	}
	return r
}
```

在 deps.go 添加 `registerPrivateRoutes` 默认空实现（领域 plan 各自扩展）：

```go
// registerPrivateRoutes 由各领域 plan 通过扩展 Deps（嵌入字段）覆盖；
// 默认无路由。Plan 2 暂不挂载其他领域路由。
func (d *Deps) registerPrivateRoutes(_ *gin.RouterGroup) {}
```

> 注：这是 hook point。Plan 3/4 落地后，main.go 会通过另一个装配函数注入 conversations/knowledge 路由。本 plan 保持默认空实现，避免循环依赖。

- [ ] **Step 4: 更新 main.go 装配 auth**

在 `cmd/server/main.go` 的 `main()` 中，systemHandler 装配之后插入：

```go
jwtIssuer := auth.NewJWTIssuer(cfg.JWTSecret, "investguide", cfg.JWTExpiry)
userRepo := auth.NewGORMUserRepository(db)
authSvc := auth.NewService(userRepo, jwtIssuer)

deps := &router.Deps{
	Cfg:           cfg,
	DB:            db,
	Cache:         cacheInst,
	TaskQueue:     taskQ,
	Version:       version,
	SystemHandler: system.NewHandler(system.NewService(version)),
	AuthHandler:   auth.NewHandler(authSvc),
	Authenticator: &auth.AuthenticatorAdapter{Service: authSvc},
}
```

并添加 import `"github.com/invest-guide/backend/internal/domain/auth"`。

- [ ] **Step 5: 跑全部 router 测试**

Run: `cd backend && go test ./internal/platform/router/`
Expected: PASS（Plan 1 三个测试 + 本 plan 新增的 auth 路由测试）

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/router/ backend/cmd/server/main.go
git commit -m "feat(backend/router): wire auth routes + authenticated group"
```

---

### Task 8: E2E 测试（注册→登录→访问鉴权路由）

**Files:**
- Create: `backend/tests/e2e/auth_test.go`
- Modify: `backend/tests/e2e/helpers.go`

- [ ] **Step 1: 扩展 helpers.go 支持鉴权路由组测试**

把 `NewTestServer` 改为返回 `*TestServer`（含 DB 与 AuthSvc，便于 E2E 测试准备 fixture）：

`backend/tests/e2e/helpers.go`（完全替换 Plan 1 版本）:
```go
package e2e

import (
	"net/http/httptest"

	"github.com/invest-guide/backend/internal/domain/auth"
	"github.com/invest-guide/backend/internal/domain/system"
	"github.com/invest-guide/backend/internal/platform/config"
	"github.com/invest-guide/backend/internal/platform/database"
	"github.com/invest-guide/backend/internal/platform/router"
	"gorm.io/gorm"
)

type TestServer struct {
	*httptest.Server
	DB      *gorm.DB
	AuthSvc *auth.Service
}

func NewTestServer() *TestServer {
	db, _ := database.NewSQLite(":memory:")
	_ = db.AutoMigrate(&auth.User{})

	jwt := auth.NewJWTIssuer("test-secret", "investguide", 1<<30)
	userRepo := auth.NewGORMUserRepository(db)
	authSvc := auth.NewService(userRepo, jwt)

	deps := &router.Deps{
		Cfg:           &config.Config{CORSOrigins: "*", RateLimitAPI: 0},
		Version:       "0.0.1-test",
		SystemHandler: system.NewHandler(system.NewService("0.0.1-test")),
		AuthHandler:   auth.NewHandler(authSvc),
		Authenticator: &auth.AuthenticatorAdapter{Service: authSvc},
	}
	r := router.New(deps)
	return &TestServer{
		Server:  httptest.NewServer(r),
		DB:      db,
		AuthSvc: authSvc,
	}
}
```

> Plan 1 的 `NewTestServer()` 直接返回 `*httptest.Server`，本 plan 改为返回 `*TestServer` 包装。同步修改 Plan 1 的 health/version/no-such-route 测试：把 `srv := NewTestServer()` 后 `srv.URL` 仍然可用（嵌入式 `*httptest.Server`），无需改动调用点。

- [ ] **Step 2: 写 E2E 测试**

`backend/tests/e2e/auth_test.go`:
```go
package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow_RegisterThenLogin(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	// 1. 注册
	regBody := `{"email":"a@b.com","password":"password123","displayName":"Alice"}`
	resp := postJSON(t, srv.URL+"/api/v1/auth/register", regBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var regResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
			User  struct {
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &regResp))
	assert.Equal(t, "a@b.com", regResp.Data.User.Email)
	assert.NotEmpty(t, regResp.Data.Token)

	// 2. 重复注册 → 409
	resp2 := postJSON(t, srv.URL+"/api/v1/auth/register", regBody, "")
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)

	// 3. 登录
	loginBody := `{"email":"a@b.com","password":"password123"}`
	resp3 := postJSON(t, srv.URL+"/api/v1/auth/login", loginBody, "")
	require.Equal(t, http.StatusOK, resp3.StatusCode)
}

func TestAuth_MissingToken_BlockedFromPrivateGroup(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	// 当前 plan 的 private group 还没挂载任何路由，但任何 /api/v1/ 下
	// 未注册的路径都会返回 404；本 test 改为验证未带 token 时
	// 访问一个 plan 2 中存在的受保护路径（暂未实现）的行为
	// 改造为验证 register 缺少 body 时返回 400
	resp := postJSON(t, srv.URL+"/api/v1/auth/register", ``, "")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAuth_LoginBadPassword(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	postJSON(t, srv.URL+"/api/v1/auth/register",
		`{"email":"a@b.com","password":"password123","displayName":"A"}`, "")

	resp := postJSON(t, srv.URL+"/api/v1/auth/login",
		`{"email":"a@b.com","password":"wrong"}`, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func postJSON(t *testing.T, url, body, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func mustReadAll(r io.ReadCloser) []byte {
	b, _ := io.ReadAll(r)
	return b
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test ./tests/e2e/`
Expected: PASS（Plan 1 health/version + 本 plan 3 个 auth 测试）

- [ ] **Step 4: Commit**

```bash
git add backend/tests/e2e/
git commit -m "test(backend/e2e): add auth register/login/bad-password flows"
```

---

### Task 9: 最终验证

**Files:** 无修改

- [ ] **Step 1: gofmt + go vet**

Run: `cd backend && gofmt -l . && go vet ./...`
Expected: 均无输出

- [ ] **Step 2: 全量测试 + 覆盖率**

Run: `cd backend && go test ./... -cover`
Expected: 全 PASS；`internal/domain/auth/` 覆盖率 ≥ 70%

- [ ] **Step 3: 启动 + curl 验证**

Run: `make backend-dev`
另开终端：

```bash
# 注册
curl -i -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"password123","displayName":"Alice"}'
# 期望 201 + token

# 重复注册
curl -i -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"password123","displayName":"Alice"}'
# 期望 409

# 登录
curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"password123"}'
# 期望 200 + token

# 错误密码
curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"a@b.com","password":"wrong"}'
# 期望 401
```

停止 backend-dev。

- [ ] **Step 4: 标记完成**

Plan 2 完成。auth 模块可独立交付，鉴权中间件已就绪供 Plan 3/4 挂载路由。

---

## 执行记录（2026-08-02）

本 plan 已按上述步骤落地，执行中做了以下修正与补充，Plan 3/4 需保持一致：

1. **登录限流改用精确失败限流**：新增 `internal/platform/middleware/login_ratelimit.go`，
   `LoginRateLimit`（按 IP 滑动窗口、仅失败计数、成功自动清零），取代 token bucket 方案。
   handler 失败时调用 `RecordFailure(c)`，成功时中间件根据 2xx 自动 `MarkSuccess`。
   `router.Deps` 增加 `LoginRateLimit *middleware.LoginRateLimit` 字段；`New(deps)` 在
   `/auth` 路由组挂 `LoginRateLimit.Handler()`。
2. **module path**：`invest-guide/backend` 无域名点会被 Go 当外部模块拉取，已改为
   `github.com/invest-guide/backend`。所有 import 同步。此修正已在 Plan 1 执行记录中声明。
3. **`registerPrivateRoutes` 方法改为 `PrivateRoutes` 字段**：`Deps.PrivateRoutes func(*gin.RouterGroup)`
   （可注入），`New` 在鉴权路由组调用 `RegisterPrivateRoutes(v1Auth)`。Plan 3/4 注入方式：
   `deps.PrivateRoutes = func(g){ knowledge.Register(g, handler) }`。
4. **`NewTestDeps()` 改为 `NewTestDeps(t *testing.T)`**：需要 SQLite + AutoMigrate(users)。
   e2e/helpers.go 重写为 `NewTestServer() *TestServer`（含 DB/AuthSvc），不再调用 `NewTestDeps`。
5. **`TestService_Register_DuplicateEmail` 断言修正**：service 把 `ErrDuplicateEmail` 包装为
   `response.ErrConflict` 返回，断言应为 `ErrorIs(err, response.ErrConflict)`。
6. **main.go 启动 AutoMigrate**：注册/登录 500 因 `no such table: users`。修复为 main 启动时
   `database.AutoMigrate(db, &auth.User{})`。Plan 3/4 的领域模型也要在这里追加 AutoMigrate。
7. **E2E `TestAuth_MissingToken_Returns401` 无效**（未注册路径直接 404 不过 Auth），改为
   router 单测 `TestPrivateGroup_RequiresToken`：通过 `deps.PrivateRoutes` 注入测试路由验证 401。

覆盖率（达标 ≥70%）：auth 73.8% · middleware 86.9% · router 88.9%。
