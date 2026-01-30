package security

import (
	"os"
	"path/filepath"
	"strings"
)

// NormalizeAndValidateDir converts path to absolute, checks it exists and is a directory,
// and rejects any path containing "..".
func NormalizeAndValidateDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if strings.Contains(abs, "..") {
		return "", os.ErrInvalid
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", os.ErrInvalid
	}
	return abs, nil
}

// IsPathAllowed returns true if absPath is under one of the allowedDirs (after cleaning).
// Rejects paths containing "..".
func IsPathAllowed(absPath string, allowedDirs []string) bool {
	cleaned := filepath.Clean(absPath)
	if strings.Contains(cleaned, "..") {
		return false
	}
	for _, d := range allowedDirs {
		canon := filepath.Clean(d)
		if canon == cleaned || strings.HasPrefix(cleaned, canon+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
