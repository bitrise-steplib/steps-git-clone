package step

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/tracker"
	"github.com/stretchr/testify/require"
)

func Test_GitCloneStep_IsCloneDirDangerous(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Safe path in temp dir",
			path:     "$TMPDIR/clone",
			expected: false,
		},
		{
			name:     "Safe path in home",
			path:     filepath.Join(home, "clone"),
			expected: false,
		},
		{
			name:     "Home as env var",
			path:     "$HOME",
			expected: true,
		},
		{
			name:     "Home as tilde",
			path:     "~",
			expected: true,
		},
		{
			name:     "Home as absolute path",
			path:     home,
			expected: true,
		},
		{
			name:     "Dangerous path with tilde",
			path:     "~/Documents",
			expected: true,
		},
		{
			name:     "Dangerous absolute path",
			path:     filepath.Join(home, ".ssh"),
			expected: true,
		},
		{
			name:     "Nonexistent env var only",
			path:     "$NONEXISTENT",
			expected: false,
		},
	}

	logger := log.NewLogger()
	envRepo := env.NewRepository()
	tracker := tracker.NewStepTracker(envRepo, logger)
	inputParser := stepconf.NewInputParser(envRepo)
	cmdFactory := command.NewFactory(envRepo)
	pathModifier := pathutil.NewPathModifier()

	gitCloneStep := NewGitCloneStep(logger, tracker, inputParser, cmdFactory, pathModifier)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := gitCloneStep.isCloneDirDangerous(test.path)
			require.Equal(t, test.expected, result)
		})
	}
}
