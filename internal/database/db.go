// internal/database/db.go
package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func New(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Run migrations
	if _, err := db.Exec(createTablesSQL); err != nil {
		return nil, err
	}

	// Seed initial categorization rules
	if _, err := db.Exec(seedCategoryRulesSQL); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}
