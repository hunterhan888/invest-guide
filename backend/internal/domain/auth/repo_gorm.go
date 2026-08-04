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
