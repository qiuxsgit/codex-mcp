package db

import (
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"
)

const directoriesDDL = `
CREATE TABLE IF NOT EXISTS directories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  path TEXT NOT NULL,
  language TEXT NOT NULL,
  role TEXT NOT NULL,
  enabled INTEGER DEFAULT 1,
  updated_at DATETIME
);
`

var conn *sql.DB

// Open opens SQLite at dbPath and runs migrations.
func Open(dbPath string) error {
	var err error
	conn, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	if _, err = conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = conn.Close()
		return err
	}
	_, err = conn.Exec(directoriesDDL)
	if err != nil {
		_ = conn.Close()
		return err
	}
	// Migrate: add git columns if missing (existing DBs)
	_ = migrateAddGitColumns()
	return nil
}

func migrateAddGitColumns() error {
	for _, q := range []string{
		`ALTER TABLE directories ADD COLUMN git_auto_update_interval_sec INTEGER DEFAULT 0`,
		`ALTER TABLE directories ADD COLUMN git_last_updated_at DATETIME`,
	} {
		_, err := conn.Exec(q)
		if err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}
	return nil
}

// DB returns the global connection (for tests or advanced use). Prefer package functions.
func DB() *sql.DB {
	return conn
}

// Close closes the database connection.
func Close() error {
	if conn == nil {
		return nil
	}
	return conn.Close()
}
