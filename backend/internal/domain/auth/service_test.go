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
	users     map[string]*User // by email
	byID      map[string]*User
	createErr error
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
	jwt := NewJWTIssuer("test-secret", "invest-guide", 3600*1_000_000_000)
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
	assert.ErrorIs(t, err, response.ErrConflict)
}

func TestService_Login_Success(t *testing.T) {
	svc, _ := newTestService()
	_, _ = svc.Register(context.Background(), RegisterRequest{
		Email: "a@b.com", Password: "password123", DisplayName: "A",
	})
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
