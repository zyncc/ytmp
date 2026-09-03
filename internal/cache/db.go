package cache

import (
	"database/sql"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const databaseFilename = "cache.db"

func DatabasePath(cacheDir string) string {
	return filepath.Join(cacheDir, databaseFilename)
}

func OpenDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", DatabasePath(path))
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
