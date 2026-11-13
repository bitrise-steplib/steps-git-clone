package gitclone

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// A committed .gitattributes with a huge conflict-marker-size overflows the heap during
// a merge (CVE-2026-105437, fixed in Git 2.56.1 and its backports).
// .git/info/attributes takes precedence over others, this pins default size for all merges
// in the repository.
// This is not the primary fix. Git is also being upgraded on the stacks, and the pin only
// protects builds that still run an unpatched Git.
// TODO: revert once all stacks ship a patched Git and self-hosted users had time to upgrade.
const conflictMarkerSizeAttribute = "* conflict-marker-size=7"

func pinConflictMarkerSize(repoDir string) error {
	path := filepath.Join(repoDir, ".git", "info", "attributes")

	content, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Later rules take precedence, so the pin is only effective as the last rule.
	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	if strings.TrimSpace(lines[len(lines)-1]) == conflictMarkerSizeAttribute {
		return nil
	}

	var updated strings.Builder
	if len(content) > 0 {
		for _, line := range lines {
			if strings.TrimSpace(line) != conflictMarkerSizeAttribute {
				updated.WriteString(line + "\n")
			}
		}
	}
	updated.WriteString(conflictMarkerSizeAttribute + "\n")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(updated.String()), 0o644)
}
