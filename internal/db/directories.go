package db

import (
	"database/sql"
	"time"

	"github.com/qiuxsgit/codex-mcp/internal/security"
)

// Directory row.
type Directory struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Language  string     `json:"language"`
	Role      string     `json:"role"`
	Enabled   bool       `json:"enabled"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// List returns all directories.
func ListDirectories() ([]Directory, error) {
	rows, err := conn.Query(`
		SELECT id, name, path, language, role, enabled, updated_at FROM directories ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Directory
	for rows.Next() {
		var d Directory
		var en int
		var uat sql.NullTime
		err = rows.Scan(&d.ID, &d.Name, &d.Path, &d.Language, &d.Role, &en, &uat)
		if err != nil {
			return nil, err
		}
		d.Enabled = en != 0
		if uat.Valid {
			t := uat.Time
			d.UpdatedAt = &t
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListEnabled returns only directories with enabled=1.
func ListEnabledDirectories() ([]Directory, error) {
	rows, err := conn.Query(`
		SELECT id, name, path, language, role, enabled, updated_at FROM directories WHERE enabled = 1 ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Directory
	for rows.Next() {
		var d Directory
		var en int
		var uat sql.NullTime
		err = rows.Scan(&d.ID, &d.Name, &d.Path, &d.Language, &d.Role, &en, &uat)
		if err != nil {
			return nil, err
		}
		d.Enabled = en != 0
		if uat.Valid {
			t := uat.Time
			d.UpdatedAt = &t
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// AddDirectory validates path (absolute, exists, no ..), then inserts.
func AddDirectory(name, path, language, role string) (int64, error) {
	absPath, err := security.NormalizeAndValidateDir(path)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	res, err := conn.Exec(`
		INSERT INTO directories (name, path, language, role, enabled, updated_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`, name, absPath, language, role, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeleteDirectory removes by id.
func DeleteDirectory(id int64) error {
	_, err := conn.Exec(`DELETE FROM directories WHERE id = ?`, id)
	return err
}

// SetEnabled sets enabled flag.
func SetDirectoryEnabled(id int64, enabled bool) error {
	en := 0
	if enabled {
		en = 1
	}
	_, err := conn.Exec(`UPDATE directories SET enabled = ?, updated_at = ? WHERE id = ?`,
		en, time.Now().UTC(), id)
	return err
}
