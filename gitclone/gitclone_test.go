package gitclone

import (
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/stretchr/testify/require"
)

func Test_checkoutState(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		want     *step.Error
		wantCmds []string
	}{
		{
			name: "Checkout commit",
			cfg: Config{
				RepositoryURL: " https://github.com/bitrise-io/steps-git-clone.git",
				Commit:        "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch"`,
				`git "checkout" "76a934ae80f12bb9b504bbc86f64a1d310e5db64"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := &MockRunner{}
			runner = mockRunner
			if got := checkoutState(git.Git{}, tt.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
			require.Equal(t, tt.wantCmds, mockRunner.Cmds())
		})
	}
}

type MockRunner struct {
	cmds []string
}

func (r *MockRunner) Cmds() []string {
	return r.cmds
}

func (r *MockRunner) Output(c *command.Model) (string, error) {
	r.cmds = append(r.cmds, c.PrintableCommandArgs())
	return "", nil
}
func (r *MockRunner) Run(c *command.Model) error {
	r.cmds = append(r.cmds, c.PrintableCommandArgs())
	return nil
}
func (r *MockRunner) RunWithRetry(c *command.Model) error {
	r.cmds = append(r.cmds, c.PrintableCommandArgs())
	return nil
}
