package gitclone

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
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
		// Commit with unshallow
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
		{
			name: "Checkout tag, branch specifed has same name as tag",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Tag:           "gat",
				Branch:        "gat",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "--tags" "origin" "refs/heads/gat"`,
				`git "checkout" "gat"`,
			},
		},
		{
			name: "Checkout PR - auto merge - merge branch (GitHub format)",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				BranchDest:    "master",
				PRMergeBranch: "pull/5/merge",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/master"`,
				`git "fetch" "origin" "refs/pull/5/head:pull/5"`,
				`git "checkout" "master"`,
				`git "merge" "origin/master"`,
				`git "merge" "pull/5"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "Checkout PR - auto merge -merge branch (standard branch format)",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				BranchDest:    "master",
				PRMergeBranch: "pr_test",
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/master"`,
				`git "fetch" "origin" "refs/heads/pr_test:pr_test"`,
				`git "checkout" "master"`,
				`git "merge" "origin/master"`,
				`git "merge" "pr_test"`, // ToDo: warning: refname 'pr_test' is ambiguous.
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "Checkout PR - auto merge - merge branch, with depth (unshallow needed)",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				BranchDest:    "master",
				PRMergeBranch: "pull/5/merge",
				CloneDepth:    1,
			},
			cmdOutputs: map[string]commandOutput{
				`git "merge" "pull/5"`: {failForCalls: 1},
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "--depth=1" "origin" "refs/heads/master"`,
				`git "fetch" "origin" "refs/pull/5/head:pull/5"`,
				`git "checkout" "master"`,
				`git "merge" "origin/master"`,
				`git "merge" "pull/5"`,
				// fatal: refusing to merge unrelated histories
				// Merge failed, error: fatal: refusing to merge unrelated histories
				`git "reset" "--hard" "HEAD"`,
				`git "clean" "-x" "-d" "-f"`,
				`git "submodule" "foreach" "git" "reset" "--hard" "HEAD"`,
				`git "submodule" "foreach" "git" "clean" "-x" "-d" "-f"`,
				`git "fetch" "--unshallow"`,
				`git "merge" "pull/5"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "UNSUPPORTED BranchDest missing -originally Checkout manual merge (ignore depth)",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Commit:        "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
				Branch:        "test/commit-messages",
				PRMergeBranch: "pull/7/merge",
				PRID:          7,
				CloneDepth:    1,
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "--depth=1" "origin" "refs/heads/"`,
				`git "fetch" "origin" "refs/pull/7/head:pull/7"`,
				`git "checkout" ""`,
				`git "merge" "origin/"`,
				`git "merge" "pull/7"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "Checkout PR - manual merge, branch and commit (ignore depth)",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/steps-git-clone.git",
				Commit:        "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
				Branch:        "test/commit-messages",
				PRMergeBranch: "pull/7/merge",
				BranchDest:    "master",
				PRID:          7,
				CloneDepth:    1,
				ManualMerge:   true,
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/master"`,
				`git "checkout" "master"`,     // Already on 'master'
				`git "merge" "origin/master"`, // Already up to date.
				`git "log" "-1" "--format=%H"`,
				`git "fetch" "origin" "refs/heads/test/commit-messages"`,
				`git "merge" "76a934ae80f12bb9b504bbc86f64a1d310e5db64"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "Checkout PR not a fork - repo is the same with different scheme - manual merge",
			cfg: Config{
				RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
				PRRepositoryURL: "git@github.com:bitrise-io/git-clone-test.git",
				Branch:          "test/commit-messages",
				BranchDest:      "master",
				PRMergeBranch:   "pull/7/merge",
				PRID:            7,
				Commit:          "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
				ManualMerge:     true,
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/master"`,
				`git "checkout" "master"`,
				`git "merge" "origin/master"`,
				`git "log" "-1" "--format=%H"`,
				`git "fetch" "origin" "refs/heads/test/commit-messages"`,
				`git "merge" "76a934ae80f12bb9b504bbc86f64a1d310e5db64"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "Checkout PR fork - manual merge",
			cfg: Config{
				RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
				PRRepositoryURL: "https://github.com/bitrise-io/other-repo.git",
				Branch:          "test/commit-messages",
				BranchDest:      "master",
				Commit:          "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
				ManualMerge:     true,
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/master"`,
				`git "checkout" "master"`,
				`git "merge" "origin/master"`,
				`git "log" "-1" "--format=%H"`,
				`git "remote" "add" "fork" "https://github.com/bitrise-io/other-repo.git"`,
				`git "fetch" "fork" "refs/heads/test/commit-messages"`,
				`git "merge" "fork/test/commit-messages"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "Checkout PR fork (private) - auto merge (overrides manual merge flag) - merge branch",
			cfg: Config{
				RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
				PRRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:          "test/commit-messages",
				BranchDest:      "master",
				PRMergeBranch:   "pull/7/merge",
				PRID:            7,
				Commit:          "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
				ManualMerge:     true,
			},
			want: nil,
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/master"`,
				`git "fetch" "origin" "refs/pull/7/head:pull/7"`,
				`git "checkout" "master"`,
				`git "merge" "origin/master"`,
				`git "merge" "pull/7"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "Checkout PR fork (private) - auto merge (overrides manual merge flag) - fails",
			cfg: Config{
				RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
				PRRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:          "test/commit-messages",
				BranchDest:      "master",
				Commit:          "76a934ae80f12bb9b504bbc86f64a1d310e5db64",
				ManualMerge:     true,
			},
			want: newStepError(
				"auto_merge_failed",
				fmt.Errorf("merging PR (automatic) failed: %v", "there is no Pull Request branch and can't download diff file"),
				"Merging pull request failed",
			),
			wantCmds: []string{
				`git "fetch" "origin" "refs/heads/master"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := newMockRunner(tt.cmdOutputs)
			runner = mockRunner
			got := checkoutState(git.Git{}, tt.cfg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkoutState().err = (%#v), want %#v", got, tt.want)
			}
			// require.Equal(t, tt.want, got)
			if !reflect.DeepEqual(mockRunner.Cmds(), tt.wantCmds) {
				t.Errorf("checkoutState().cmds =\n%v, want\n%v", mockRunner.Cmds(), tt.wantCmds)
			}
			// require.Equal(t, tt.wantCmds, mockRunner.Cmds())
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
