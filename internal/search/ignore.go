package search

import (
	"path/filepath"
	"strings"
)

// fixed patterns always applied (like .git, node_modules, target, vendor)
var fixedIgnoreDirs = map[string]bool{
	".git": true, "node_modules": true, "target": true, "vendor": true,
}

// IgnoreRules parses gitignore-style content into a list of patterns.
// Each non-empty, non-comment line is a pattern (glob or path segment).
type IgnoreRules struct {
	patterns []string // trimmed, non-empty, not comment
}

// ParseIgnoreRules parses raw gitignore-style content.
func ParseIgnoreRules(data []byte) *IgnoreRules {
	r := &IgnoreRules{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r.patterns = append(r.patterns, line)
	}
	return r
}

// ShouldIgnore returns true if path (relative to search root or absolute) should be ignored.
// isDir: true for directory, false for file.
// Supports: "name", "path/name", "*.ext", "*/.name".
func (r *IgnoreRules) ShouldIgnore(path string, isDir bool) bool {
	path = filepath.ToSlash(path)
	base := filepath.Base(path)

	// Fixed dirs: any path segment matching .git, node_modules, target, vendor
	for _, part := range strings.Split(path, "/") {
		if fixedIgnoreDirs[part] {
			return true
		}
	}

	for _, pat := range r.patterns {
		pat = filepath.ToSlash(strings.TrimSpace(pat))
		if pat == "" {
			continue
		}
		// Glob: *.ext, *.*, etc.
		if strings.Contains(pat, "*") {
			ok, _ := filepath.Match(pat, base)
			if ok {
				return true
			}
			// Also try matching full path suffix (e.g. "**/*.log" -> match "*.log" on base)
			if strings.HasPrefix(pat, "**/") {
				ok, _ = filepath.Match(pat[3:], base)
				if ok {
					return true
				}
			}
			continue
		}
		// Dir pattern: "dir" or "dir/" -> path contains "dir" as segment
		if strings.HasSuffix(pat, "/") {
			pat = pat[:len(pat)-1]
		}
		if strings.Contains(path, "/"+pat+"/") || strings.HasSuffix(path, "/"+pat) || path == pat || base == pat {
			return true
		}
	}
	return false
}
