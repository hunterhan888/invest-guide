package auth

import (
	"context"
	"errors"
)

var ErrDuplicateEmail = errors.New("duplicate email")

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
}
