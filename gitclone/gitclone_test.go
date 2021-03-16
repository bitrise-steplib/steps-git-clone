package gitclone

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/stretchr/testify/assert"
)

const rawCmdError = "dummy_cmd_error"

var testCases = [...]struct {
	name        string
	cfg         Config
	patchSource patchSource
	mockRunner  *MockRunner
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
			Commit:     "76a934a",
			CloneDepth: 1,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules"`,
			`git "checkout" "76a934a"`,
		},
	},
	{
		name: "Checkout commit, branch specified",
		cfg: Config{
			Commit:    "76a934ae",
			Branch:    "hcnarb",
			FetchTags: true,
		},
		wantCmds: []string{
			`git "fetch" "--tags" "--no-recurse-submodules" "origin" "refs/heads/hcnarb"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "Checkout commit with retry",
		cfg: Config{
			Commit: "76a934ae",
		},
		mockRunner: givenMockRunnerSucceedsAfter(1),
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "Checkout branch",
		cfg: Config{
			Branch:     "hcnarb",
			CloneDepth: 1,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/hcnarb"`,
			`git "checkout" "hcnarb"`,
			`git "merge" "origin/hcnarb"`,
		},
	},
	{
		name: "Checkout tag",
		cfg: Config{
			Tag:        "gat",
			CloneDepth: 1,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/tags/gat:refs/tags/gat"`,
			`git "checkout" "gat"`,
		},
	},
	{
		name: "Checkout tag, branch specifed",
		cfg: Config{
			Tag: "gat",
		},
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/tags/gat:refs/tags/gat"`,
			`git "checkout" "gat"`,
		},
	},
	{
		name: "Checkout tag, branch specifed has same name as tag",
		cfg: Config{
			Tag: "gat",
		},
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/tags/gat:refs/tags/gat"`,
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
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/hcnarb"`,
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
			`git "fetch" "--no-tags" "--no-recurse-submodules"`,
			`git "checkout" "76a934ae"`,
		},
	},

	// ** PRs manual merge
	{
		name: "PR - no fork - manual merge: branch and commit",
		cfg: Config{
			Commit:        "76a934ae",
			Branch:        "test/commit-messages",
			PRMergeBranch: "pull/7/merge",
			PRDestBranch:  "master",
			CloneDepth:    1,
			ManualMerge:   true,
			ShouldMergePR: true,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,     // Already on 'master'
			`git "merge" "origin/master"`, // Already up to date.
			`git "log" "-1" "--format=%H"`,
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/test/commit-messages"`,
			`git "merge" "76a934ae"`,
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - no fork - manual merge: branch and commit, no PRRepoURL or PRID",
		cfg: Config{
			Commit:        "76a934ae",
			Branch:        "test/commit-messages",
			PRDestBranch:  "master",
			ManualMerge:   true,
			ShouldMergePR: true,
		},
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "log" "-1" "--format=%H"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/test/commit-messages"`,
			`git "merge" "76a934ae"`,
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - fork - manual merge",
		cfg: Config{
			RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
			PRSourceRepositoryURL: "https://github.com/bitrise-io/other-repo.git",
			Branch:                "test/commit-messages",
			PRDestBranch:          "master",
			Commit:                "76a934ae",
			CloneDepth:            1,
			ManualMerge:           true,
			ShouldMergePR:         true,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "log" "-1" "--format=%H"`,
			`git "remote" "add" "fork" "https://github.com/bitrise-io/other-repo.git"`,
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "fork" "refs/heads/test/commit-messages"`,
			`git "merge" "fork/test/commit-messages"`,
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - no fork - manual merge: repo is the same with different scheme",
		cfg: Config{
			RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
			PRSourceRepositoryURL: "git@github.com:bitrise-io/git-clone-test.git",
			Branch:                "test/commit-messages",
			PRDestBranch:          "master",
			PRMergeBranch:         "pull/7/merge",
			Commit:                "76a934ae",
			ManualMerge:           true,
			ShouldMergePR:         true,
		},
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "log" "-1" "--format=%H"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/test/commit-messages"`,
			`git "merge" "76a934ae"`,
			`git "checkout" "--detach"`,
		},
	},

	// ** PRs auto merge **
	{
		name: "PR - no fork - auto merge - merge branch (GitHub format)",
		cfg: Config{
			PRDestBranch:  "master",
			PRMergeBranch: "pull/5/merge",
			PRHeadBranch:  "pull/5/head",
			CloneDepth:    1,
			ShouldMergePR: true,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/5/head:pull/5"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "merge" "pull/5"`,
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - no fork - auto merge - merge branch (standard branch format)",
		cfg: Config{
			PRDestBranch:  "master",
			PRMergeBranch: "pr_test",
			ShouldMergePR: true,
		},
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/pr_test:pr_test"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "merge" "pr_test"`, // warning: refname 'pr_test' is ambiguous.
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - fork - auto merge - merge branch: private fork overrides manual merge flag",
		cfg: Config{
			RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
			PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
			Branch:                "test/commit-messages",
			PRDestBranch:          "master",
			PRMergeBranch:         "pull/7/merge",
			Commit:                "76a934ae",
			ManualMerge:           true,
			ShouldMergePR:         true,
		},
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/7/head:pull/7"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "merge" "pull/7"`,
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - fork - auto merge - diff file: private fork overrides manual merge flag, Fails",
		cfg: Config{
			RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
			PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
			Branch:                "test/commit-messages",
			PRDestBranch:          "master",
			Commit:                "76a934ae",
			ManualMerge:           true,
			ShouldMergePR:         true,
			BuildURL:              "dummy_url",
			UpdateSubmodules:      true,
		},
		patchSource: MockPatchSource{"", errors.New(rawCmdError)},
		mockRunner: givenMockRunner().
			GivenRunWithRetryFailsAfter(2).
			GivenRunSucceeds(),
		wantCmds: []string{
			`git "fetch" "--no-tags" "origin" "refs/heads/master"`,
			`git "fetch" "--no-tags" "origin" "refs/heads/master"`,
			`git "fetch" "--no-tags" "origin" "refs/heads/master"`,
			`git "fetch"`,
			`git "branch" "-r"`,
		},
		wantErrType: &step.Error{},
	},
	{
		name: "PR - fork - auto merge - diff file: private fork overrides manual merge flag",
		cfg: Config{
			RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
			Branch:        "test/commit-messages",
			PRDestBranch:  "master",
			Commit:        "76a934ae",
			CloneDepth:    1,
			ManualMerge:   false,
			ShouldMergePR: true,
			BuildURL:      "dummy_url",
		},
		patchSource: MockPatchSource{"diff_path", nil},
		wantErr:     nil,
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "apply" "--index" "diff_path"`,
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - no fork - auto merge - diff file: fallback to manual merge if unable to apply patch",
		cfg: Config{
			RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
			Branch:        "test/commit-messages",
			PRDestBranch:  "master",
			Commit:        "76a934ae",
			CloneDepth:    1,
			ManualMerge:   false,
			ShouldMergePR: true,
			BuildURL:      "dummy_url",
		},
		patchSource: MockPatchSource{"diff_path", nil},
		mockRunner: givenMockRunner().
			GivenRunFailsForCommand(`git "apply" "--index" "diff_path"`, 1).
			GivenRunWithRetrySucceeds().
			GivenRunSucceeds(),
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "apply" "--index" "diff_path"`,
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "log" "-1" "--format=%H"`,
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/test/commit-messages"`,
			`git "merge" "76a934ae"`,
			`git "checkout" "--detach"`,
		},
	},
	{
		name: "PR - fork - auto merge - diff file: fallback to manual merge if unable to apply patch",
		cfg: Config{
			RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
			PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
			Branch:                "test/commit-messages",
			PRDestBranch:          "master",
			Commit:                "76a934ae",
			ManualMerge:           true,
			ShouldMergePR:         true,
			BuildURL:              "dummy_url",
		},
		patchSource: MockPatchSource{"diff_path", nil},
		mockRunner: givenMockRunner().
			GivenRunFailsForCommand(`git "apply" "--index" "diff_path"`, 1).
			GivenRunWithRetrySucceeds().
			GivenRunSucceeds(),
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "apply" "--index" "diff_path"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "checkout" "master"`,
			`git "merge" "origin/master"`,
			`git "log" "-1" "--format=%H"`,
			`git "remote" "add" "fork" "git@github.com:bitrise-io/other-repo.git"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules" "fork" "refs/heads/test/commit-messages"`,
			`git "merge" "fork/test/commit-messages"`,
			`git "checkout" "--detach"`,
		},
	},

	// PRs no merge
	{
		name: "PR - no merge - no fork - manual merge: branch and commit",
		cfg: Config{
			Commit:           "76a934ae",
			Branch:           "test/commit-messages",
			PRDestBranch:     "master",
			CloneDepth:       1,
			ManualMerge:      true,
			ShouldMergePR:    false,
			UpdateSubmodules: true,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "origin" "refs/heads/test/commit-messages"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "PR - no merge - no fork - auto merge - head branch",
		cfg: Config{
			Commit:           "76a934ae",
			PRDestBranch:     "master",
			PRMergeBranch:    "pull/5/merge",
			PRHeadBranch:     "pull/5/head",
			CloneDepth:       1,
			ShouldMergePR:    false,
			UpdateSubmodules: true,
		},
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "origin" "refs/pull/5/head"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "PR - no merge - no fork - auto merge - diff file: public fork",
		cfg: Config{
			RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
			PRSourceRepositoryURL: "https://github.com/bitrise-io/git-clone-test2.git",
			Branch:                "test/commit-messages",
			PRDestBranch:          "master",
			Commit:                "76a934ae",
			CloneDepth:            1,
			ManualMerge:           false,
			ShouldMergePR:         false,
			UpdateSubmodules:      true,
		},
		patchSource: MockPatchSource{"diff_path", nil},
		wantErr:     nil,
		wantCmds: []string{
			`git "remote" "add" "fork" "https://github.com/bitrise-io/git-clone-test2.git"`,
			`git "fetch" "--depth=1" "--no-tags" "fork" "refs/heads/test/commit-messages"`,
			`git "checkout" "76a934ae"`,
		},
	},
	{
		name: "PR - no merge - fork - auto merge - diff file: private fork",
		cfg: Config{
			RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
			PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
			Branch:                "test/commit-messages",
			PRDestBranch:          "master",
			Commit:                "76a934ae",
			CloneDepth:            1,
			ManualMerge:           false,
			ShouldMergePR:         false,
			UpdateSubmodules:      true,
			BuildURL:              "dummy_url",
		},
		patchSource: MockPatchSource{"diff_path", nil},
		wantErr:     nil,
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "origin" "refs/heads/master"`,
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
		mockRunner: givenMockRunner().
			GivenRunWithRetryFailsAfter(2).
			GivenRunSucceeds(),
		wantCmds: []string{
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/fake"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/fake"`,
			`git "fetch" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/fake"`,
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
			CloneDepth:    1,
			ShouldMergePR: true,
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
		mockRunner: givenMockRunner().
			GivenRunFailsForCommand(`git "checkout" "cfba2b01332e31cb1568dbf3f22edce063118bae"`, 1).
			GivenRunWithRetrySucceeds().
			GivenRunSucceeds(),
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags"`,
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
			PRDestBranch:  "master",
			ManualMerge:   true,
			ShouldMergePR: true,
		},
		wantErrType: ParameterValidationError{},
		wantCmds:    nil,
	},
	{
		name: "Checkout PR - auto merge - merge branch, with depth (unshallow needed)",
		cfg: Config{
			PRDestBranch:  "master",
			PRMergeBranch: "pull/5/merge",
			PRHeadBranch:  "pull/5/head",
			CloneDepth:    1,
			ShouldMergePR: true,
		},
		mockRunner: givenMockRunner().
			GivenRunFailsForCommand(`git "merge" "pull/5"`, 1).
			GivenRunSucceeds().
			GivenRunWithRetrySucceeds(),
		wantCmds: []string{
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
			`git "fetch" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/5/head:pull/5"`,
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
			// Given
			var mockRunner *MockRunner
			if tt.mockRunner != nil {
				mockRunner = tt.mockRunner
			} else {
				mockRunner = givenMockRunnerSucceeds()
			}
			runner = mockRunner

			// When
			actualErr := checkoutState(git.Git{}, tt.cfg, tt.patchSource)

			// Then
			if tt.wantErrType != nil {
				assert.IsType(t, tt.wantErrType, actualErr)
			} else if tt.wantErr != nil {
				assert.EqualError(t, actualErr, tt.wantErr.Error())
			} else {
				assert.Nil(t, actualErr)
			}

			assert.Equal(t, tt.wantCmds, mockRunner.Cmds())
		})
	}
}

// SubmoduleUpdate
var submoduleTestCases = [...]struct {
	name     string
	cfg      Config
	wantCmds []string
}{
	{
		name: "Limiting submodule depth",
		cfg: Config{
			LimitSubmoduleUpdateDepth: true,
		},
		wantCmds: []string{
			`git "submodule" "update" "--init" "--recursive" "--depth=1"`,
		},
	},
	{
		name: "Not limiting submodule depth",
		cfg: Config{
			LimitSubmoduleUpdateDepth: false,
		},
		wantCmds: []string{
			`git "submodule" "update" "--init" "--recursive"`,
		},
	},
}

func Test_SubmoduleUpdate(t *testing.T) {
	for _, tt := range submoduleTestCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRunner := givenMockRunnerSucceeds()
			runner = mockRunner

			// When
			actualErr := updateSubmodules(git.Git{}, tt.cfg)

			// Then
			assert.NoError(t, actualErr)
			assert.Equal(t, tt.wantCmds, mockRunner.Cmds())
		})
	}
}

// Mocks
func givenMockRunner() *MockRunner {
	mockRunner := new(MockRunner)
	mockRunner.GivenRunForOutputSucceeds()
	return mockRunner
}

func givenMockRunnerSucceeds() *MockRunner {
	return givenMockRunnerSucceedsAfter(0)
}

func givenMockRunnerSucceedsAfter(times int) *MockRunner {
	return givenMockRunner().
		GivenRunWithRetrySucceedsAfter(times).
		GivenRunSucceeds()
}

type MockPatchSource struct {
	diffFilePath string
	err          error
}

func (m MockPatchSource) getDiffPath(_, _ string) (string, error) {
	return m.diffFilePath, m.err
}
