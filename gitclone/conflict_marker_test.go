package gitclone

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_pinConflictMarkerSize(t *testing.T) {
	tests := []struct {
		name     string
		existing *string
		want     string
	}{
		{
			name: "No attributes file",
			want: "* conflict-marker-size=7\n",
		},
		{
			name:     "Keeps existing attributes",
			existing: ptr("*.png binary"),
			want:     "*.png binary\n* conflict-marker-size=7\n",
		},
		{
			name:     "Overrides an earlier size",
			existing: ptr("* conflict-marker-size=99999999\n"),
			want:     "* conflict-marker-size=99999999\n* conflict-marker-size=7\n",
		},
		{
			name:     "Moves an overridden pin last",
			existing: ptr("* conflict-marker-size=7\n* conflict-marker-size=99999999\n"),
			want:     "* conflict-marker-size=99999999\n* conflict-marker-size=7\n",
		},
		{
			name:     "Already pinned",
			existing: ptr("* conflict-marker-size=7\n"),
			want:     "* conflict-marker-size=7\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := t.TempDir()
			path := filepath.Join(repoDir, ".git", "info", "attributes")
			if tt.existing != nil {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(*tt.existing), 0o644))
			}

			require.NoError(t, pinConflictMarkerSize(repoDir))
			require.NoError(t, pinConflictMarkerSize(repoDir))

			got, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}

func Test_pinConflictMarkerSize_overridesInTreeAttributes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "sub", ".gitattributes"), []byte("* conflict-marker-size=99999999\n"), 0o644))

	require.Equal(t, "sub/file.txt: conflict-marker-size: 99999999", runGit(t, repoDir, "check-attr", "conflict-marker-size", "--", "sub/file.txt"))

	require.NoError(t, pinConflictMarkerSize(repoDir))

	require.Equal(t, "sub/file.txt: conflict-marker-size: 7", runGit(t, repoDir, "check-attr", "conflict-marker-size", "--", "sub/file.txt"))
}

func runGit(t *testing.T, dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return strings.TrimSpace(string(out))
}

func ptr(s string) *string {
	return &s
}
