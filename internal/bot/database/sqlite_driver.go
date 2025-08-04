package database

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// NewSQLiteDialector создает новый диалект SQLite для GORM без CGO
func NewSQLiteDialector(dsn string) gorm.Dialector {
	return sqlite.Open(dsn)
}
