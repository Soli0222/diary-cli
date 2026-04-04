package git

import (
	"fmt"
	"os/exec"
	"strings"
)

var run = runCommand

// CommitAndPush stages, commits, and pushes the given file.
func CommitAndPush(repoDir, filePath, date string) error {
	if err := run(repoDir, "git", "add", "--", filePath); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	commitMsg := fmt.Sprintf("diary: %s", date)
	if err := run(repoDir, "git", "commit", "-m", commitMsg, "--only", "--", filePath); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	if err := run(repoDir, "git", "push"); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	return nil
}

func runCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
