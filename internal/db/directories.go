package db

import (
	"database/sql"
	"time"

	"github.com/qiuxsgit/codex-mcp/internal/security"
)

// Valid directory roles: 前端业务, 后端业务, 前端框架, 后端框架.
// MCP 搜索时可用 role 参数限定为「前端」或「后端」。
var ValidRoles = []string{"前端业务", "后端业务", "前端框架", "后端框架"}

// IsValidRole returns true if role is one of ValidRoles.
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// Directory row.
type Directory struct {
	ID                         int64      `json:"id"`
	Name                       string     `json:"name"`
	Path                       string     `json:"path"`
	Language                   string     `json:"language"`
	Role                       string     `json:"role"`
	Enabled                    bool       `json:"enabled"`
	UpdatedAt                  *time.Time `json:"updated_at,omitempty"`
	GitAutoUpdateIntervalSec   int        `json:"git_auto_update_interval_sec"`
	GitLastUpdatedAt           *time.Time `json:"git_last_updated_at,omitempty"`
}

// List returns all directories.
func ListDirectories() ([]Directory, error) {
	rows, err := conn.Query(`
		SELECT id, name, path, language, role, enabled, updated_at,
		       COALESCE(git_auto_update_interval_sec, 0), git_last_updated_at
		FROM directories ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Directory
	for rows.Next() {
		var d Directory
		var en int
		var uat, glat sql.NullTime
		err = rows.Scan(&d.ID, &d.Name, &d.Path, &d.Language, &d.Role, &en, &uat, &d.GitAutoUpdateIntervalSec, &glat)
		if err != nil {
			return nil, err
		}
		d.Enabled = en != 0
		if uat.Valid {
			t := uat.Time
			d.UpdatedAt = &t
		}
		if glat.Valid {
			t := glat.Time
			d.GitLastUpdatedAt = &t
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListEnabled returns only directories with enabled=1.
func ListEnabledDirectories() ([]Directory, error) {
	rows, err := conn.Query(`
		SELECT id, name, path, language, role, enabled, updated_at,
		       COALESCE(git_auto_update_interval_sec, 0), git_last_updated_at
		FROM directories WHERE enabled = 1 ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Directory
	for rows.Next() {
		var d Directory
		var en int
		var uat, glat sql.NullTime
		err = rows.Scan(&d.ID, &d.Name, &d.Path, &d.Language, &d.Role, &en, &uat, &d.GitAutoUpdateIntervalSec, &glat)
		if err != nil {
			return nil, err
		}
		d.Enabled = en != 0
		if uat.Valid {
			t := uat.Time
			d.UpdatedAt = &t
		}
		if glat.Valid {
			t := glat.Time
			d.GitLastUpdatedAt = &t
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

// GetDirectoryByID returns a directory by id, or nil if not found.
func GetDirectoryByID(id int64) (*Directory, error) {
	row := conn.QueryRow(`
		SELECT id, name, path, language, role, enabled, updated_at,
		       COALESCE(git_auto_update_interval_sec, 0), git_last_updated_at
		FROM directories WHERE id = ?
	`, id)
	var d Directory
	var en int
	var uat, glat sql.NullTime
	err := row.Scan(&d.ID, &d.Name, &d.Path, &d.Language, &d.Role, &en, &uat, &d.GitAutoUpdateIntervalSec, &glat)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	d.Enabled = en != 0
	if uat.Valid {
		t := uat.Time
		d.UpdatedAt = &t
	}
	if glat.Valid {
		t := glat.Time
		d.GitLastUpdatedAt = &t
	}
	return &d, nil
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

// SetDirectoryGitInterval sets git auto-update interval in seconds (0 = disabled).
func SetDirectoryGitInterval(id int64, intervalSec int) error {
	if intervalSec < 0 {
		intervalSec = 0
	}
	_, err := conn.Exec(`UPDATE directories SET git_auto_update_interval_sec = ?, updated_at = ? WHERE id = ?`,
		intervalSec, time.Now().UTC(), id)
	return err
}

// UpdateDirectoryGitLastUpdated sets git_last_updated_at for a directory.
func UpdateDirectoryGitLastUpdated(id int64, t time.Time) error {
	_, err := conn.Exec(`UPDATE directories SET git_last_updated_at = ?, updated_at = ? WHERE id = ?`,
		t.UTC(), time.Now().UTC(), id)
	return err
}

// ListDirectoriesForGitUpdate returns directories with git_auto_update_interval_sec > 0 that are due for update
// (last_updated is null or last_updated + interval <= now).
func ListDirectoriesForGitUpdate(now time.Time) ([]Directory, error) {
	rows, err := conn.Query(`
		SELECT id, name, path, language, role, enabled, updated_at,
		       COALESCE(git_auto_update_interval_sec, 0), git_last_updated_at
		FROM directories
		WHERE COALESCE(git_auto_update_interval_sec, 0) > 0
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Directory
	for rows.Next() {
		var d Directory
		var en int
		var uat, glat sql.NullTime
		err = rows.Scan(&d.ID, &d.Name, &d.Path, &d.Language, &d.Role, &en, &uat, &d.GitAutoUpdateIntervalSec, &glat)
		if err != nil {
			return nil, err
		}
		d.Enabled = en != 0
		if uat.Valid {
			t := uat.Time
			d.UpdatedAt = &t
		}
		if glat.Valid {
			t := glat.Time
			d.GitLastUpdatedAt = &t
		}
		due := d.GitLastUpdatedAt == nil
		if !due {
			next := d.GitLastUpdatedAt.Add(time.Duration(d.GitAutoUpdateIntervalSec) * time.Second)
			due = !now.Before(next)
		}
		if due {
			out = append(out, d)
		}
	}
	return out, rows.Err()
}
