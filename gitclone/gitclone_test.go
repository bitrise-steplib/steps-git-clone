package gitclone

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
)

const always = 9999
const rawCmdError = "dummy_cmd_error"

var testCases = [...]struct {
	name        string
	cfg         Config
	patchSource patchSource
	cmdOutputs  map[string]commandOutput
	wantErr     error
	wantErrType error
	wantCmds    []string
}{
	// ** Simple checkout cases (using commit, tag and branch) **
	{
		name:     "No checkout args",
		cfg:      Config{},
		wantCmds: nil,
	},
	{
		name: "Checkout commit",
		cfg: Config{
			Commit: "76a934a",
		},
		wantCmds: []string{
			`git "fetch"`,
			`git "checkout" "76a934a"`,
		},
	},
	{
		name: "Checkout commit, branch specified",
		cfg: Config{
			Commit: "76a934ae",
			Branch: "hcnarb",
		},
		wantCmds: []string{
			`git "fetch"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "Checkout commit with retry",
		cfg: Config{
			Commit: "76a934ae",
		},
		cmdOutputs: map[string]commandOutput{
			`git "fetch"`: {failForCalls: 1},
		},
		wantCmds: []string{
			`git "fetch"`,
			`git "fetch"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "Checkout branch",
		cfg: Config{
			Branch: "hcnarb",
		},
		wantCmds: []string{
			`git "fetch" "origin" "refs/heads/hcnarb"`,
			`git "checkout" "hcnarb"`,
			`git "merge" "origin/hcnarb"`,
		},
	},
	{
		name: "Checkout tag",
		cfg: Config{
			Tag: "gat",
		},
		wantCmds: []string{
			`git "fetch" "--tags"`,
			`git "checkout" "gat"`,
		},
	},
	{
		name: "Checkout tag, branch specifed",
		cfg: Config{
			Tag:    "gat",
			Branch: "hcnarb",
		},
		wantCmds: []string{
			`git "fetch" "--tags" "origin" "refs/heads/hcnarb"`,
			`git "checkout" "gat"`,
		},
	},
	{
		name: "Checkout tag, branch specifed has same name as tag",
		cfg: Config{
			Tag:    "gat",
			Branch: "gat",
		},
		wantCmds: []string{
			`git "fetch" "--tags" "origin" "refs/heads/gat"`,
			`git "checkout" "gat"`,
		},
	},
	{
		name: "UNSUPPORTED Checkout commit, tag, branch specifed",
		cfg: Config{
			Commit: "76a934ae",
			Tag:    "gat",
			Branch: "hcnarb",
		},
		wantCmds: []string{
			`git "fetch" "--tags"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "UNSUPPORTED Checkout commit, tag specifed",
		cfg: Config{
			Commit: "76a934ae",
			Tag:    "gat",
		},
		wantCmds: []string{
			`git "fetch" "--tags"`,
			`git "checkout" "76a934ae"`,
		},
	},

	// ** PRs manual merge
	{
		name: "PR - no fork - manual merge: branch and commit (ignore depth)",
		cfg: Config{
			Commit:        "76a934ae",
			Branch:        "test/commit-messages",
			PRMergeBranch: "pull/7/merge",
			BranchDest:    "master",
			PRID:          7,
			CloneDepth:    1,
			ManualMerge:   true,
		},
		wantErr: nil,
		wantCmds: []string{
			`git "fetch" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,     // Already on 'master'
			`git "merge" "origin/master"`, // Already up to date.
			`git "log" "-1" "--format=%H"`,
			`git "fetch" "origin" "refs/heads/test/commit-messages"`,
			`git "merge" "76a934ae"`,
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - no fork - manual merge: branch and commit (Checked out as commit, why?)",
		cfg: Config{
			Commit:      "76a934ae",
			Branch:      "test/commit-messages",
			BranchDest:  "master",
			CloneDepth:  1,
			ManualMerge: true,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "PR - fork - manual merge",
		cfg: Config{
			RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
			PRRepositoryURL: "https://github.com/bitrise-io/other-repo.git",
			Branch:          "test/commit-messages",
			BranchDest:      "master",
			Commit:          "76a934ae",
			ManualMerge:     true,
		},
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
		name: "PR - no fork - manual merge: repo is the same with different scheme",
		cfg: Config{
			RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
			PRRepositoryURL: "git@github.com:bitrise-io/git-clone-test.git",
			Branch:          "test/commit-messages",
			BranchDest:      "master",
			PRMergeBranch:   "pull/7/merge",
			PRID:            7,
			Commit:          "76a934ae",
			ManualMerge:     true,
		},
		wantCmds: []string{
			`git "fetch" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "log" "-1" "--format=%H"`,
			`git "fetch" "origin" "refs/heads/test/commit-messages"`,
			`git "merge" "76a934ae"`,
			`git "checkout" "--detach"`,
		},
	},

	// ** PRs auto merge **
	{
		name: "PR - no fork - auto merge - merge branch (GitHub format)",
		cfg: Config{
			BranchDest:    "master",
			PRMergeBranch: "pull/5/merge",
		},
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
		name: "PR - no fork - auto merge - merge branch (standard branch format)",
		cfg: Config{
			BranchDest:    "master",
			PRMergeBranch: "pr_test",
		},
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
		name: "PR - fork - auto merge - merge branch: private fork overrides manual merge flag",
		cfg: Config{
			RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
			PRRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
			Branch:          "test/commit-messages",
			BranchDest:      "master",
			PRMergeBranch:   "pull/7/merge",
			PRID:            7,
			Commit:          "76a934ae",
			ManualMerge:     true,
		},
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
		name: "PR - fork - auto merge - diff file: private fork overrides manual merge flag, Fails",
		cfg: Config{
			RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
			PRRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
			Branch:          "test/commit-messages",
			BranchDest:      "master",
			Commit:          "76a934ae",
			ManualMerge:     true,
		},
		patchSource: MockPatchSource{"", errors.New(rawCmdError)},
		wantErr:     fmt.Errorf("merging PR (automatic) failed, there is no Pull Request branch and could not download diff file: %s", rawCmdError),
		wantCmds:    nil,
	},
	{
		name: "PR - fork - auto merge - diff file: private fork overrides manual merge flag",
		cfg: Config{
			RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
			Branch:        "test/commit-messages",
			BranchDest:    "master",
			PRID:          7,
			ManualMerge:   false,
		},
		patchSource: MockPatchSource{"diff_path", nil},
		wantErr:     nil,
		wantCmds: []string{
			`git "fetch" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "apply" "--index" "diff_path"`,
			`git "checkout" "--detach"`,
		},
	},

	// ** Errors **
	{
		name: "Checkout nonexistent branch",
		cfg: Config{
			Branch: "fake",
		},
		cmdOutputs: map[string]commandOutput{
			`git "fetch" "origin" "refs/heads/fake"`: {failForCalls: always},
			`git "branch" "-r"`:                      {output: "  origin/master"}, //"ToDO:  origin/master\n  origin/HEAD -> origin/master"
		},
		wantCmds: []string{
			`git "fetch" "origin" "refs/heads/fake"`,
			`git "fetch" "origin" "refs/heads/fake"`,
			`git "fetch" "origin" "refs/heads/fake"`,
			`git "fetch"`,
			`git "branch" "-r"`,
		},
		wantErr: newStepErrorWithBranchRecommendations(
			fetchFailedTag,
			fmt.Errorf("fetch failed: %v", errors.New(rawCmdError)),
			"Fetching repository has failed",
			"fake",
			[]string{"master"},
		),
	},
	{
		name: "PR - no fork - auto merge: BranchDest missing (UNSUPPORTED)",
		cfg: Config{
			Commit:        "76a934ae",
			Branch:        "test/commit-messages",
			PRMergeBranch: "pull/7/merge",
			PRID:          7,
			CloneDepth:    1,
		},
		wantErrType: ParameterValidationError{},
		wantCmds:    nil,
	},

	// ** CloneDepth specified, Unshallow needed **
	{
		name: "Checkout commit, unshallow needed",
		cfg: Config{
			Commit:           "cfba2b01332e31cb1568dbf3f22edce063118bae",
			CloneDepth:       1,
			UpdateSubmodules: true,
		},
		cmdOutputs: map[string]commandOutput{
			`git "checkout" "cfba2b01332e31cb1568dbf3f22edce063118bae"`: {failForCalls: 1},
		},
		wantCmds: []string{
			`git "fetch" "--depth=1"`,
			`git "checkout" "cfba2b01332e31cb1568dbf3f22edce063118bae"`,
			// fatal: reference is not a tree: cfba2b01332e31cb1568dbf3f22edce063118bae
			// Checkout failed, error: fatal: reference is not a tree: cfba2b01332e31cb1568dbf3f22edce063118bae
			`git "fetch" "--unshallow"`,
			`git "checkout" "cfba2b01332e31cb1568dbf3f22edce063118bae"`,
		},
	},
	{
		name: "PR - no fork - manual merge: branch, no commit (ignore depth) UNSUPPORTED?",
		cfg: Config{
			Branch:        "test/commit-messages",
			PRMergeBranch: "pull/7/merge",
			BranchDest:    "master",
			ManualMerge:   true,
		},
		wantErrType: ParameterValidationError{},
		wantCmds:    nil,
	},
	{
		name: "Checkout PR - auto merge - merge branch, with depth (unshallow needed)",
		cfg: Config{
			BranchDest:    "master",
			PRMergeBranch: "pull/5/merge",
			CloneDepth:    1,
		},
		cmdOutputs: map[string]commandOutput{
			`git "merge" "pull/5"`: {failForCalls: 1},
		},
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
}

func Test_checkoutState(t *testing.T) {
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := newMockRunner(tt.cmdOutputs)
			runner = mockRunner
			gotErr := checkoutState(git.Git{}, tt.cfg, tt.patchSource)

			if tt.wantErrType != nil {
				if reflect.TypeOf(tt.wantErrType) != reflect.TypeOf(gotErr) {
					t.Errorf("checkoutState().err type = (%T), want (%T)", gotErr, tt.wantErrType)
				}
			} else {
				if !reflect.DeepEqual(gotErr, tt.wantErr) {
					t.Errorf("checkoutState().err = (%#v), want %#v", gotErr, tt.wantErr)
				}
			}

			if !reflect.DeepEqual(mockRunner.Cmds(), tt.wantCmds) {
				t.Errorf("checkoutState().cmds =\n%v, want\n%v", mockRunner.Cmds(), tt.wantCmds)
			}
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

func (r *MockRunner) RunForOutput(c *command.Model) (string, error) {
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
	_, err := r.RunForOutput(c)
	return err
}

func (r *MockRunner) RunWithRetry(getCommnad func() *command.Model) error {
	var err error
	for i := 0; i < 3; i++ {
		err = r.Run(getCommnad())
		if err == nil {
			return nil
		}
	}

	return err
}

type MockPatchSource struct {
	diffFilePath string
	err          error
}

func (m MockPatchSource) getDiffPath(_, _ string) (string, error) {
	return m.diffFilePath, m.err
}
