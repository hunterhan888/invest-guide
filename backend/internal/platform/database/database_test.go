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

	assert.NoError(t, db.Exec("CREATE TABLE probe (id INTEGER PRIMARY KEY)").Error)
}

func TestNewSQLite_PersistsData(t *testing.T) {
	db, err := NewSQLite(":memory:")
	assert.NoError(t, err)
	assert.NoError(t, db.Exec("CREATE TABLE note (text TEXT)").Error)
	assert.NoError(t, db.Exec("INSERT INTO note (text) VALUES ('hello')").Error)

	var got string
	assert.NoError(t, db.Raw("SELECT text FROM note").Scan(&got).Error)
	assert.Equal(t, "hello", got)
}

func TestConnect_SQLite(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.NoError(t, db.Exec("SELECT 1").Error)
}

func TestConnect_UnsupportedScheme(t *testing.T) {
	_, err := Connect("mysql://localhost/db")
	assert.Error(t, err)
}

func TestTransactionWrap_Commit(t *testing.T) {
	db, _ := NewSQLite(":memory:")
	assert.NoError(t, db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error)

	err := TransactionWrap(db, func(tx *gorm.DB) error {
		return tx.Exec("INSERT INTO t (v) VALUES ('ok')").Error
	})
	assert.NoError(t, err)

	var count int
	assert.NoError(t, db.Raw("SELECT COUNT(*) FROM t").Scan(&count).Error)
	assert.Equal(t, 1, count)
}

func TestTransactionWrap_Rollback(t *testing.T) {
	db, _ := NewSQLite(":memory:")
	assert.NoError(t, db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error)

	err := TransactionWrap(db, func(tx *gorm.DB) error {
		assert.NoError(t, tx.Exec("INSERT INTO t (v) VALUES ('a')").Error)
		return assert.AnError
	})
	assert.Error(t, err)

	var count int
	assert.NoError(t, db.Raw("SELECT COUNT(*) FROM t").Scan(&count).Error)
	assert.Equal(t, 0, count) // 事务回滚
}
