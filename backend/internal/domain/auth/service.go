package auth

import (
	"context"
	"errors"

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
