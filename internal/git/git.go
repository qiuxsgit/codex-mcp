package git

import (
	"os"
	"os/exec"
	"path/filepath"
)

// IsGitRepo returns true if path is the root of a git repository.
func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Pull runs "git pull" in path. Path must be a git repo root.
func Pull(path string) error {
	cmd := exec.Command("git", "-C", path, "pull", "--ff-only")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
