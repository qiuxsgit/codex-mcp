package search

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/qiuxsgit/codex-mcp/internal/config"
	"github.com/qiuxsgit/codex-mcp/internal/db"
	"github.com/qiuxsgit/codex-mcp/internal/security"
)

const (
	maxSnippetLines = 15
	maxResponseKB   = 50
)

var rgWarnOnce sync.Once

// RgAvailable returns true if ripgrep (rg) is installed and on PATH.
func RgAvailable() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

// Match is one search result.
type Match struct {
	Path        string `json:"path"`
	LineStart   int    `json:"line_start"`
	LineEnd     int    `json:"line_end"`
	Snippet     string `json:"snippet"`
	MatchReason string `json:"match_reason"`
}

// Params for search.
type Params struct {
	Query      string // 搜索关键词
	Language   string // 可选语言过滤
	PathHint   string // 可选路径子串
	Role       string // 可选范围：前端 / 后端，只搜对应角色的目录
	Limit      int
	IgnorePath string
}

// Search runs search in enabled directories. If ripgrep (rg) is installed, uses rg for better performance; otherwise falls back to built-in pure Go search and logs a one-time hint to install rg.
func Search(p Params) ([]Match, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 20 {
		p.Limit = 20
	}

	dirs, err := db.ListEnabledDirectories()
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return []Match{}, nil
	}

	// 按角色过滤：前端 -> 前端业务/前端框架，后端 -> 后端业务/后端框架
	if p.Role != "" {
		var filtered []db.Directory
		for _, d := range dirs {
			switch p.Role {
			case "前端":
				if d.Role == "前端业务" || d.Role == "前端框架" {
					filtered = append(filtered, d)
				}
			case "后端":
				if d.Role == "后端业务" || d.Role == "后端框架" {
					filtered = append(filtered, d)
				}
			default:
				filtered = append(filtered, d)
			}
		}
		dirs = filtered
	}
	if len(dirs) == 0 {
		return []Match{}, nil
	}

	var searchDirs []string
	for _, d := range dirs {
		if p.PathHint != "" && !strings.Contains(d.Path, p.PathHint) {
			continue
		}
		searchDirs = append(searchDirs, d.Path)
	}
	if len(searchDirs) == 0 {
		return []Match{}, nil
	}

	allowedPaths := make([]string, len(searchDirs))
	for i := range searchDirs {
		allowedPaths[i] = filepath.Clean(searchDirs[i])
	}

	if RgAvailable() {
		return searchWithRg(p, searchDirs, allowedPaths)
	}
	rgWarnOnce.Do(func() {
		log.Printf("[search] ripgrep (rg) 未安装，使用内置搜索。建议安装 rg 以提升搜索性能: https://github.com/BurntSushi/ripgrep#installation")
	})
	return searchBuiltin(p, searchDirs, allowedPaths)
}

// searchWithRg runs ripgrep in searchDirs and parses output into matches.
func searchWithRg(p Params, searchDirs, allowedPaths []string) ([]Match, error) {
	args := []string{
		"-n",
		"--no-heading",
		"-g", "!.git",
		"-g", "!node_modules",
		"-g", "!target",
		"-g", "!vendor",
	}
	if p.IgnorePath != "" {
		_, _ = config.ReadIgnoreFile(p.IgnorePath)
		args = append(args, "--ignore-file", p.IgnorePath)
	}
	if p.Language != "" {
		args = append(args, "-t", strings.ToLower(p.Language))
	}
	args = append(args, regexp.QuoteMeta(p.Query))
	args = append(args, searchDirs...)

	cmd := exec.Command("rg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 1 {
			return nil, err
		}
		// rg exits 1 when no match; ignore
	}

	linePattern := regexp.MustCompile(`^(.+?):(\d+):(.*)`)
	var matches []Match
	scanner := bufio.NewScanner(&stdout)
	totalBytes := 0
	maxBytes := maxResponseKB * 1024

	for scanner.Scan() && len(matches) < p.Limit && totalBytes < maxBytes {
		line := scanner.Text()
		sub := linePattern.FindStringSubmatch(line)
		if sub == nil {
			continue
		}
		path, lineStr, content := sub[1], sub[2], sub[3]
		path = filepath.Clean(path)
		if !security.IsPathAllowed(path, allowedPaths) {
			continue
		}
		lineNum, err := strconv.Atoi(lineStr)
		if err != nil {
			continue
		}
		snippet := buildSnippet(path, lineNum, content, maxSnippetLines)
		if snippet == "" {
			continue
		}
		lines := strings.Split(snippet, "\n")
		start, end := lineNum, lineNum
		if len(lines) > 1 {
			half := (len(lines) - 1) / 2
			start = lineNum - half
			end = lineNum + (len(lines) - 1 - half)
			if start < 1 {
				start = 1
			}
		}
		m := Match{
			Path:        path,
			LineStart:   start,
			LineEnd:     end,
			Snippet:     snippet,
			MatchReason: "content",
		}
		matches = append(matches, m)
		totalBytes += len(path) + len(snippet) + 64
	}
	return matches, nil
}

// searchBuiltin runs pure Go (WalkDir + regex) search.
func searchBuiltin(p Params, searchDirs, allowedPaths []string) ([]Match, error) {
	var rules *IgnoreRules
	if p.IgnorePath != "" {
		data, _ := config.ReadIgnoreFile(p.IgnorePath)
		if len(data) > 0 {
			rules = ParseIgnoreRules(data)
		}
	}
	if rules == nil {
		rules = ParseIgnoreRules(nil)
	}

	pattern := regexp.QuoteMeta(p.Query)
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return nil, err
	}

	extFilter := languageToExt(p.Language)
	maxBytes := maxResponseKB * 1024
	var matches []Match
	var totalBytes int

	for _, root := range searchDirs {
		if len(matches) >= p.Limit || totalBytes >= maxBytes {
			break
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if len(matches) >= p.Limit || totalBytes >= maxBytes {
				return filepath.SkipAll
			}
			path = filepath.Clean(path)
			if !security.IsPathAllowed(path, allowedPaths) {
				return filepath.SkipDir
			}
			if d.IsDir() {
				if rules.ShouldIgnore(path, true) {
					return filepath.SkipDir
				}
				return nil
			}
			if rules.ShouldIgnore(path, false) {
				return nil
			}
			if extFilter != "" && !strings.HasSuffix(strings.ToLower(path), extFilter) {
				return nil
			}
			fileMatches := searchFile(path, re, p.Limit-len(matches), maxBytes-totalBytes)
			for _, m := range fileMatches {
				matches = append(matches, m)
				totalBytes += len(m.Path) + len(m.Snippet) + 64
			}
			return nil
		})
	}
	return matches, nil
}

func languageToExt(lang string) string {
	switch strings.ToLower(lang) {
	case "go":
		return ".go"
	case "rust", "rs":
		return ".rs"
	case "python", "py":
		return ".py"
	case "javascript", "js":
		return ".js"
	case "typescript", "ts":
		return ".ts"
	case "java":
		return ".java"
	case "c":
		return ".c"
	case "cpp", "c++":
		return ".cpp"
	case "ruby", "rb":
		return ".rb"
	case "php":
		return ".php"
	case "swift":
		return ".swift"
	case "kotlin", "kt":
		return ".kt"
	case "scala":
		return ".scala"
	case "sh", "shell", "bash":
		return ".sh"
	case "html":
		return ".html"
	case "css":
		return ".css"
	case "json":
		return ".json"
	case "yaml", "yml":
		return ".yml"
	default:
		return ""
	}
}

func searchFile(filePath string, re *regexp.Regexp, maxMatches int, maxBytes int) []Match {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var matches []Match
	sc := bufio.NewScanner(f)
	lineNum := 0
	fileBytes := 0

	for sc.Scan() && len(matches) < maxMatches && fileBytes < maxBytes {
		lineNum++
		line := sc.Text()
		if !re.MatchString(line) {
			continue
		}
		snippet := buildSnippet(filePath, lineNum, line, maxSnippetLines)
		if snippet == "" {
			continue
		}
		lines := strings.Split(snippet, "\n")
		start, end := lineNum, lineNum
		if len(lines) > 1 {
			half := (len(lines) - 1) / 2
			start = lineNum - half
			end = lineNum + (len(lines) - 1 - half)
			if start < 1 {
				start = 1
			}
		}
		m := Match{
			Path:        filePath,
			LineStart:   start,
			LineEnd:     end,
			Snippet:     snippet,
			MatchReason: "content",
		}
		matches = append(matches, m)
		fileBytes += len(filePath) + len(snippet) + 64
	}

	return matches
}

func buildSnippet(filePath string, matchLine int, matchContent string, maxLines int) string {
	f, err := os.Open(filePath)
	if err != nil {
		return strings.TrimSpace(matchContent)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		if lineNum > matchLine+maxLines/2 {
			break
		}
		if lineNum >= matchLine-maxLines/2 {
			lines = append(lines, sc.Text())
		}
	}
	if sc.Err() != nil {
		return strings.TrimSpace(matchContent)
	}
	if len(lines) == 0 {
		return strings.TrimSpace(matchContent)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}
