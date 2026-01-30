package db

import (
	"database/sql"

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
