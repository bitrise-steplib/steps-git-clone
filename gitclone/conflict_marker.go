package gitclone

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// A committed .gitattributes with a huge conflict-marker-size overflows the heap during
// a merge (CVE-2026-105437, fixed in Git 2.56.1 and the backports in conflictMarkerFixedPatch).
// .git/info/attributes takes precedence over others, this pins default size for all merges
// in the repository.
const conflictMarkerSizeAttribute = "* conflict-marker-size=7"

// First patch release with the CVE-2026-105437 fix per minor version. Older minors got no fix.
var conflictMarkerFixedPatch = map[int]int{50: 2, 51: 3, 52: 1, 53: 1, 54: 1, 55: 1, 56: 1}

var gitVersionPattern = regexp.MustCompile(`git version (\d+)\.(\d+)\.(\d+)`)

// The pin also overrides sizes that repos set on purpose, so it's only written for vulnerable or unknown versions.
func (g GitCloner) pinConflictMarkerSizeIfVulnerable(repoDir string) {
	out, err := g.cmdFactory.Create("git", []string{"version"}, nil).RunAndReturnTrimmedOutput()
	if err == nil && hasConflictMarkerFix(out) {
		return
	}
	if err := pinConflictMarkerSize(repoDir); err != nil {
		g.logger.Warnf("Failed to pin conflict-marker-size in .git/info/attributes: %s", err)
	}
}

func hasConflictMarkerFix(gitVersionOutput string) bool {
	match := gitVersionPattern.FindStringSubmatch(gitVersionOutput)
	if match == nil {
		return false
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])

	if major != 2 {
		return major > 2
	}
	if minor > 56 {
		return true
	}
	fixedPatch, ok := conflictMarkerFixedPatch[minor]
	return ok && patch >= fixedPatch
}

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
