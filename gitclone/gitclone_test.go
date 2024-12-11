package gitclone

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/bitriseapi"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/tracker"
	"github.com/stretchr/testify/assert"
)

const rawCmdError = "dummy_cmd_error"

func Test_checkoutState(t *testing.T) {
	var tests = [...]struct {
		name            string
		cfg             Config
		patchSource     bitriseapi.PatchSource
		mergeRefChecker bitriseapi.MergeRefChecker
		mockRunner      *MockRunner
		wantErr         error
		wantErrType     error
		wantCmds        []string
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
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules"`,
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
				`git "fetch" "--jobs=10" "--depth=1" "--tags" "--no-recurse-submodules" "origin" "refs/heads/hcnarb"`,
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
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules"`,
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
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/hcnarb"`,
				`git "checkout" "-B" "hcnarb" "origin/hcnarb"`,
			},
		},
		{
			name: "Checkout tag",
			cfg: Config{
				Tag:        "gat",
				CloneDepth: 1,
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/tags/gat:refs/tags/gat"`,
				`git "checkout" "gat"`,
			},
		},
		{
			name: "Checkout tag, branch specified",
			cfg: Config{
				Tag: "gat",
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/tags/gat:refs/tags/gat"`,
				`git "checkout" "gat"`,
			},
		},
		{
			name: "Checkout tag, branch specified has same name as tag",
			cfg: Config{
				Tag: "gat",
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/tags/gat:refs/tags/gat"`,
				`git "checkout" "gat"`,
			},
		},
		{
			name: "UNSUPPORTED Checkout commit, tag, branch specified",
			cfg: Config{
				Commit: "76a934ae",
				Tag:    "gat",
				Branch: "hcnarb",
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/hcnarb"`,
				`git "checkout" "76a934ae"`,
			},
		},
		{
			name: "UNSUPPORTED Checkout commit, tag specified",
			cfg: Config{
				Commit: "76a934ae",
				Tag:    "gat",
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules"`,
				`git "checkout" "76a934ae"`,
			},
		},

		// ** PRs **
		{
			name: "PR - no fork - merge ref (GitHub format)",
			cfg: Config{
				PRDestBranch:  "master",
				PRMergeRef:    "pull/5/merge",
				PRHeadBranch:  "pull/5/head",
				CloneDepth:    1,
				ShouldMergePR: true,
			},
			wantCmds: []string{
				`git "update-ref" "-d" "refs/remotes/pull/5/merge"`,
				`git "update-ref" "-d" "refs/remotes/pull/5/head"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/5/merge:refs/remotes/pull/5/merge"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/5/head:refs/remotes/pull/5/head"`,
				`git "checkout" "refs/remotes/pull/5/merge"`,
			},
		},
		{
			name: "PR - no fork - merge ref (standard branch format)",
			cfg: Config{
				PRDestBranch:  "master",
				PRMergeRef:    "pr_test",
				ShouldMergePR: true,
			},
			wantErrType: ParameterValidationError{},
		},
		{
			name: "PR - fork - merge ref: private fork",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				PRMergeRef:            "pull/7/merge",
				PRHeadBranch:          "pull/7/head",
				Commit:                "76a934ae",
				ShouldMergePR:         true,
			},
			wantCmds: []string{
				`git "update-ref" "-d" "refs/remotes/pull/7/merge"`,
				`git "update-ref" "-d" "refs/remotes/pull/7/head"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/7/merge:refs/remotes/pull/7/merge"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/7/head:refs/remotes/pull/7/head"`,
				`git "checkout" "refs/remotes/pull/7/merge"`,
			},
		},
		{
			name: "PR - fork - diff file: private fork overrides manual merge flag, Fails",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ShouldMergePR:         true,
				UpdateSubmodules:      true,
			},
			patchSource: FakePatchSource{"", errors.New(rawCmdError)},
			mockRunner: givenMockRunner().
				GivenRunWithRetryFailsAfter(2).
				GivenRunSucceeds(),
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=50" "--no-tags" "origin" "refs/heads/master"`,
				`git "fetch" "--jobs=10" "--depth=50" "--no-tags" "origin" "refs/heads/master"`,
				`git "fetch" "--jobs=10" "--depth=50" "--no-tags" "origin" "refs/heads/master"`,
				`git "fetch" "--jobs=10"`,
				`git "branch" "-r"`,
			},
			wantErr: fmt.Errorf("failed to fetch base branch: fetch branch refs/heads/master: dummy_cmd_error: please make sure the branch still exists"),
		},
		{
			name: "PR - fork - no merge ref - diff file available",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
				Branch:        "test/commit-messages",
				PRDestBranch:  "master",
				Commit:        "76a934ae",
				CloneDepth:    1,
				ShouldMergePR: true,
			},
			patchSource: FakePatchSource{"diff_path", nil},
			wantErr:     nil,
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
				`git "checkout" "master"`,
				`git "apply" "--index" "diff_path"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "PR - no fork - diff file: fallback to manual merge if unable to apply patch",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
				Branch:        "test/commit-messages",
				PRDestBranch:  "master",
				Commit:        "76a934ae",
				CloneDepth:    1,
				ShouldMergePR: true,
			},
			patchSource: FakePatchSource{"diff_path", nil},
			mockRunner: givenMockRunner().
				GivenRunFailsForCommand(`git "apply" "--index" "diff_path"`, 1).
				GivenRunWithRetrySucceeds().
				GivenRunSucceeds(),
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
				`git "checkout" "master"`,
				`git "apply" "--index" "diff_path"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
				`git "checkout" "-B" "master" "origin/master"`,
				`git "log" "-1" "--format=%H"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/test/commit-messages"`,
				`git "merge" "76a934ae"`,
				`git "checkout" "--detach"`,
			},
		},
		{
			name: "PR - fork - diff file: fallback to manual merge if unable to apply patch",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ShouldMergePR:         true,
			},
			patchSource: FakePatchSource{"diff_path", nil},
			mockRunner: givenMockRunner().
				GivenRunFailsForCommand(`git "apply" "--index" "diff_path"`, 1).
				GivenRunWithRetrySucceeds().
				GivenRunSucceeds(),
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
				`git "checkout" "master"`,
				`git "apply" "--index" "diff_path"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/master"`,
				`git "checkout" "-B" "master" "origin/master"`,
				`git "log" "-1" "--format=%H"`,
				`git "remote" "add" "fork" "git@github.com:bitrise-io/other-repo.git"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "fork" "refs/heads/test/commit-messages"`,
				`git "merge" "fork/test/commit-messages"`,
				`git "checkout" "--detach"`,
			},
		},

		// PRs no merge
		{
			name: "PR - no merge - no fork: branch and commit",
			cfg: Config{
				Commit:           "76a934ae",
				Branch:           "test/commit-messages",
				PRDestBranch:     "master",
				CloneDepth:       1,
				ShouldMergePR:    false,
				UpdateSubmodules: true,
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "origin" "refs/heads/test/commit-messages"`,
				`git "checkout" "76a934ae"`,
			},
		},
		{
			name: "PR - no merge - no fork - merge ref - head branch",
			cfg: Config{
				Commit:           "76a934ae",
				PRDestBranch:     "master",
				PRMergeRef:       "pull/5/merge",
				PRHeadBranch:     "pull/5/head",
				CloneDepth:       1,
				ShouldMergePR:    false,
				UpdateSubmodules: true,
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "origin" "refs/pull/5/head"`,
				`git "checkout" "76a934ae"`,
			},
		},
		{
			name: "PR - no merge - no fork - diff file: public fork",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "https://github.com/bitrise-io/git-clone-test2.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				CloneDepth:            1,
				ShouldMergePR:         false,
				UpdateSubmodules:      true,
			},
			patchSource: FakePatchSource{"diff_path", nil},
			wantErr:     nil,
			wantCmds: []string{
				`git "remote" "add" "fork" "https://github.com/bitrise-io/git-clone-test2.git"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "fork" "refs/heads/test/commit-messages"`,
				`git "checkout" "76a934ae"`,
			},
		},
		{
			name: "PR - no merge - fork - diff file: private fork",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				CloneDepth:            1,
				ShouldMergePR:         false,
				UpdateSubmodules:      true,
			},
			patchSource: FakePatchSource{"diff_path", nil},
			wantErr:     nil,
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "origin" "refs/heads/master"`,
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
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/fake"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/fake"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/fake"`,
				`git "fetch" "--jobs=10"`,
				`git "branch" "-r"`,
			},
			wantErr: newStepErrorWithBranchRecommendations(
				fetchFailedTag,
				fmt.Errorf("fetch branch refs/heads/fake: %w: please make sure the branch still exists", errors.New(rawCmdError)),
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
				PRMergeRef:    "pull/7/merge",
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
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags"`,
				`git "checkout" "cfba2b01332e31cb1568dbf3f22edce063118bae"`,
				// fatal: reference is not a tree: cfba2b01332e31cb1568dbf3f22edce063118bae
				// Checkout failed, error: fatal: reference is not a tree: cfba2b01332e31cb1568dbf3f22edce063118bae
				`git "fetch" "--jobs=10" "--unshallow" "--no-tags"`,
				`git "checkout" "cfba2b01332e31cb1568dbf3f22edce063118bae"`,
			},
		},
		{
			name: "PR - no fork: branch, no commit (ignore depth)",
			cfg: Config{
				Branch:        "test/commit-messages",
				PRMergeRef:    "pull/7/merge",
				PRHeadBranch:  "pull/7/head",
				PRDestBranch:  "master",
				ShouldMergePR: true,
			},
			wantCmds: []string{
				`git "update-ref" "-d" "refs/remotes/pull/7/merge"`,
				`git "update-ref" "-d" "refs/remotes/pull/7/head"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/7/merge:refs/remotes/pull/7/merge"`,
				`git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/pull/7/head:refs/remotes/pull/7/head"`,
				`git "checkout" "refs/remotes/pull/7/merge"`,
			},
		},
		// ** Sparse-checkout **
		{
			name: "Checkout commit - sparse",
			cfg: Config{
				Commit:            "76a934a",
				CloneDepth:        1,
				SparseDirectories: []string{"client/android"},
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--filter=tree:0" "--no-tags" "--no-recurse-submodules"`,
				`git "checkout" "76a934a"`,
			},
		},
		{
			name: "Checkout commit, branch specified - sparse",
			cfg: Config{
				Commit:            "76a934ae",
				Branch:            "hcnarb",
				SparseDirectories: []string{"client/android"},
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--filter=tree:0" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/hcnarb"`,
				`git "checkout" "76a934ae"`,
			},
		},
		{
			name: "Checkout branch - sparse",
			cfg: Config{
				Branch:            "hcnarb",
				CloneDepth:        1,
				SparseDirectories: []string{"client/android"},
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--filter=tree:0" "--no-tags" "--no-recurse-submodules" "origin" "refs/heads/hcnarb"`,
				`git "checkout" "-B" "hcnarb" "origin/hcnarb"`,
			},
		},
		{
			name: "Checkout tag - sparse",
			cfg: Config{
				Tag:               "gat",
				CloneDepth:        1,
				SparseDirectories: []string{"client/android"},
			},
			wantCmds: []string{
				`git "fetch" "--jobs=10" "--depth=1" "--filter=tree:0" "--no-tags" "--no-recurse-submodules" "origin" "refs/tags/gat:refs/tags/gat"`,
				`git "checkout" "gat"`,
			},
		},
	}

	for _, tt := range tests {
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
			envRepo := env.NewRepository()
			logger := log.NewLogger()
			tracker := tracker.NewStepTracker(envRepo, logger)
			cloner := NewGitCloner(log.NewLogger(), tracker, command.NewFactory(envRepo), tt.patchSource, tt.mergeRefChecker)
			_, _, actualErr := cloner.checkoutState(git.Git{}, tt.cfg)

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

func Test_SubmoduleUpdate(t *testing.T) {
	var tests = [...]struct {
		name     string
		cfg      Config
		wantCmds []string
	}{
		{
			name: "Given submodule update depth is 1 when the submodules are updated then expect the --depth=1 flag on the command",
			cfg:  Config{SubmoduleUpdateDepth: 1},
			wantCmds: []string{
				`git "submodule" "update" "--init" "--recursive" "--jobs=10" "--depth=1"`,
			},
		},
		{
			name: "Given submodule update depth is 10 when the submodules are updated then expect the --depth=10 flag on the command",
			cfg:  Config{SubmoduleUpdateDepth: 10},
			wantCmds: []string{
				`git "submodule" "update" "--init" "--recursive" "--jobs=10" "--depth=10"`,
			},
		},
		{
			name: "Given no submodule update depth is provided when the submodules are updated then expect the --depth flag missing from the command",
			wantCmds: []string{
				`git "submodule" "update" "--init" "--recursive" "--jobs=10"`,
			},
		},
		{
			name: "Given submodule update depth is 0 when the submodules are updated then expect the --depth flag missing from the command",
			cfg:  Config{SubmoduleUpdateDepth: 0},
			wantCmds: []string{
				`git "submodule" "update" "--init" "--recursive" "--jobs=10"`,
			},
		},
		{
			name: "Given submodule update depth is -1 when the submodules are updated then expect the --depth flag missing from the command",
			cfg:  Config{SubmoduleUpdateDepth: -1},
			wantCmds: []string{
				`git "submodule" "update" "--init" "--recursive" "--jobs=10"`,
			},
		},
	}

	for _, tt := range tests {
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

func Test_SetupSparseCheckout(t *testing.T) {
	var tests = [...]struct {
		name              string
		sparseDirectories []string
		wantCmds          []string
	}{
		{
			name:              "Sparse-checkout single directory",
			sparseDirectories: []string{"client/android"},
			wantCmds: []string{
				`git "sparse-checkout" "init" "--cone"`,
				`git "sparse-checkout" "set" "client/android"`,
				`git "config" "extensions.partialClone" "origin" "--local"`,
			},
		},
		{
			name:              "Sparse-checkout multiple directory",
			sparseDirectories: []string{"client/android", "client/ios"},
			wantCmds: []string{
				`git "sparse-checkout" "init" "--cone"`,
				`git "sparse-checkout" "set" "client/android" "client/ios"`,
				`git "config" "extensions.partialClone" "origin" "--local"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRunner := givenMockRunnerSucceeds()
			runner = mockRunner

			// When
			actualErr := setupSparseCheckout(git.Git{}, tt.sparseDirectories)

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

type FakePatchSource struct {
	diffFilePath string
	err          error
}

func (f FakePatchSource) GetPRPatch() (string, error) {
	return f.diffFilePath, f.err
}

type FakeMergeRefChecker struct {
	isUpToDate bool
	err        error
}

func (f FakeMergeRefChecker) IsMergeRefUpToDate(ref string) (bool, error) {
	return f.isUpToDate, f.err
}
