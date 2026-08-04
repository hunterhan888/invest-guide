package auth

import (
	"context"
	"testing"

	"github.com/invest-guide/backend/internal/platform/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}))
	return db
}

func TestGORMRepo_CreateAndFindByEmail(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewGORMUserRepository(db)

	u := &User{ID: "u-1", Email: "x@b.com", PasswordHash: "hash", DisplayName: "X"}
	require.NoError(t, repo.Create(context.Background(), u))

	got, err := repo.FindByEmail(context.Background(), "x@b.com")
	require.NoError(t, err)
	assert.Equal(t, "u-1", got.ID)
	assert.Equal(t, "hash", got.PasswordHash)

	byID, err := repo.FindByID(context.Background(), "u-1")
	require.NoError(t, err)
	assert.Equal(t, "x@b.com", byID.Email)
}

func TestGORMRepo_DuplicateEmail(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewGORMUserRepository(db)

	u1 := &User{ID: "u-1", Email: "dup@b.com", PasswordHash: "h1"}
	require.NoError(t, repo.Create(context.Background(), u1))

	u2 := &User{ID: "u-2", Email: "dup@b.com", PasswordHash: "h2"}
	err := repo.Create(context.Background(), u2)
	assert.ErrorIs(t, err, ErrDuplicateEmail)
}

func TestGORMRepo_NotFound(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewGORMUserRepository(db)

	_, err := repo.FindByEmail(context.Background(), "nope@b.com")
	assert.ErrorIs(t, err, response.ErrNotFound)

	_, err = repo.FindByID(context.Background(), "missing")
	assert.ErrorIs(t, err, response.ErrNotFound)
}
