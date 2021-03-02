package gitclone

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/stretchr/testify/require"
)

const always = 9999
const rawCmdError = "dummy_cmd_error"

func Test_checkoutState(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		cmdOutputs map[string]commandOutput
		want       *step.Error
		wantCmds   []string
	}{
		{
			name: "Checkout commit",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Commit:        "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch"`,
				`git "checkout" "76a934ae80f12bb9b504bbc86f64a1d310e5db64"`,
			},
		},
		{
			name: "Checkout commit, branch specified",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Commit:        "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
				Branch:        "hcnarb",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch"`,
				`git "checkout" "76a934ae80f12bb9b504bbc86f64a1d310e5db64"`,
			},
		},
		{
			name: "Checkout commit with retry",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Commit:        "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
			},
			cmdOutputs: map[string]commandOutput{
				`git "fetch"`: {failForCalls: 1},
			},
			want: nil,
			wantCmds: []string{
				`git "fetch"`,
				`git "fetch"`,
				`git "checkout" "76a934ae80f12bb9b504bbc86f64a1d310e5db64"`,
			},
		},
		{
			name: "Checkout branch",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Branch:        "hcnarb",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/hcnarb"`,
				`git "checkout" "hcnarb"`,
				`git "merge" "origin/hcnarb"`,
			},
		},
		{
			name: "Checkout nonexistent branch",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Branch:        "fake",
			},
			cmdOutputs: map[string]commandOutput{
				`git "fetch" "origin" "refs/heads/fake"`: {failForCalls: always},
				`git "branch" "-r"`:                      {output: "  origin/master"}, //"  origin/master\n  origin/HEAD -> origin/master"
			},
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/fake"`,
				`git "fetch" "origin" "refs/heads/fake"`,
				`git "fetch" "origin" "refs/heads/fake"`,
				`git "fetch"`,
				`git "branch" "-r"`,
			},
			want: newStepErrorWithBranchRecommendations(
				fetchFailedTag,
				fmt.Errorf("fetch failed, error: %v", errors.New(rawCmdError)),
				"Fetching repository has failed",
				"fake",
				[]string{"master"},
			),
		},
		{
			name: "Checkout tag",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Tag:           "gat",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "--tags"`,
				`git "checkout" "gat"`,
			},
		},
		{
			name: "Checkout tag, branch specifed",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Tag:           "gat",
				Branch:        "hcnarb",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "--tags" "origin" "refs/heads/hcnarb"`,
				`git "checkout" "gat"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := newMockRunner(tt.cmdOutputs)
			runner = mockRunner
			got := checkoutState(git.Git{}, tt.cfg)
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("checkoutState().err = (%#v), want %v", got, tt.want)
			// }
			require.Equal(t, tt.want, got)
			// if !reflect.DeepEqual(mockRunner.Cmds(), tt.wantCmds) {
			// 	t.Errorf("checkoutState().cmds =\n%v, want\n%v", mockRunner.Cmds(), tt.wantCmds)
			// }
			require.Equal(t, tt.wantCmds, mockRunner.Cmds())
		})
	}
}

type commandOutput struct {
	output       string
	failForCalls int
}

type MockRunner struct {
	cmds       []string
	cmdOutputs map[string]commandOutput
}

func newMockRunner(cmdOutputs map[string]commandOutput) *MockRunner {
	return &MockRunner{cmdOutputs: cmdOutputs}
}

func (r *MockRunner) Cmds() []string {
	return r.cmds
}

func (r *MockRunner) Output(c *command.Model) (string, error) {
	commandID := c.PrintableCommandArgs()

	count := 0
	for _, cmd := range r.cmds {
		if cmd == commandID {
			count = count + 1
		}
	}

	r.cmds = append(r.cmds, commandID)

	if r.cmdOutputs == nil {
		return "", nil
	}
	if cmdOutput, ok := r.cmdOutputs[commandID]; ok {
		if cmdOutput.failForCalls != 0 && count < cmdOutput.failForCalls {
			return "", errors.New(rawCmdError)
		}
		return cmdOutput.output, nil
	}

	return "", nil
}

func (r *MockRunner) Run(c *command.Model) error {
	_, err := r.Output(c)
	return err
}

func (r *MockRunner) RunWithRetry(c *command.Model) error {
	var err error
	for i := 0; i < 3; i++ {
		err = r.Run(c)
		if err == nil {
			return nil
		}
	}

	return err
}
