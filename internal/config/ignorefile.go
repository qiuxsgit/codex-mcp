package config

import (
	"os"
	"path/filepath"
)

// DefaultIgnoreContent is the initial content for the ignore file when it does not exist.
const DefaultIgnoreContent = `# codex-mcp ignore rules (gitignore format)
# Directories
.git
node_modules
target
vendor
# Files
*.log
*.tmp
*.temp
.DS_Store
`

// ReadIgnoreFile returns raw content of the ignore file (gitignore format).
// If the file does not exist, creates it with DefaultIgnoreContent and returns that content.
func ReadIgnoreFile(path string) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			defaultContent := []byte(DefaultIgnoreContent)
			if wErr := WriteIgnoreFile(path, defaultContent); wErr != nil {
				return defaultContent, nil // return default content even if write failed (e.g. read-only dir)
			}
			return defaultContent, nil
		}
		return nil, err
	}
	return data, nil
}

// WriteIgnoreFile writes content to the ignore file. Creates parent dir if needed.
func WriteIgnoreFile(path string, content []byte) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, content, 0644)
}
