package cache

import (
	"database/sql"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func OpenDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", filepath.Join(path, "cache.db"))
	if err != nil {
		return nil, err

	}

	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
