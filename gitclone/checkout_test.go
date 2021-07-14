package gitclone

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_selectCheckoutMethod(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		patchSource patchSource
		want        CheckoutMethod
	}{
		{
			name: "none",
			cfg:  Config{},
			want: CheckoutNoneMethod,
		},
		{
			name: "commit",
			cfg: Config{
				Commit: "76a934a",
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "commit + branch",
			cfg: Config{
				Commit: "76a934ae",
				Branch: "hcnarb",
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "branch",
			cfg: Config{
				Branch: "hcnarb",
			},
			want: CheckoutBranchMethod,
		},
		{
			name: "tag",
			cfg: Config{
				Tag: "gat",
			},
			want: CheckoutTagMethod,
		},
		{
			name: "Checkout tag, branch specifed",
			cfg: Config{
				Tag:    "gat",
				Branch: "hcnarb",
			},
			want: CheckoutTagMethod,
		},
		{
			name: "UNSUPPORTED Checkout commit, tag, branch specifed",
			cfg: Config{
				Commit: "76a934ae",
				Tag:    "gat",
				Branch: "hcnarb",
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "UNSUPPORTED Checkout commit, tag specifed",
			cfg: Config{
				Commit: "76a934ae",
				Tag:    "gat",
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "PR - no fork - manual merge: branch and commit",
			cfg: Config{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRMergeBranch: "pull/7/merge",
				PRDestBranch:  "master",
				PRID:          7,
				CloneDepth:    1,
				ManualMerge:   true,
				ShouldMergePR: true,
			},
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - manual merge: branch and commit, no PRRepoURL or PRID",
			cfg: Config{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ManualMerge:   true,
				ShouldMergePR: true,
			},
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - fork - manual merge",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "https://github.com/bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         true,
			},
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - manual merge: repo is the same with different scheme",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/git-clone-test.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				PRMergeBranch:         "pull/7/merge",
				PRID:                  7,
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         true,
			},
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - auto merge - merge branch (GitHub format)",
			cfg: Config{
				PRDestBranch:  "master",
				PRMergeBranch: "pull/5/merge",
				ShouldMergePR: true,
			},
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - no fork - auto merge - diff file",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
				PRDestBranch:  "master",
				PRID:          7,
				Commit:        "76a934ae",
				ShouldMergePR: true,
				BuildURL:      "dummy_url",
			},
			patchSource: MockPatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutPRDiffFileMethod,
		},
		{
			name: "PR - fork - auto merge - merge branch: private fork overrides manual merge flag",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				PRMergeBranch:         "pull/7/merge",
				PRID:                  7,
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         true,
			},
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - fork - auto merge: private fork overrides manual merge flag",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				BuildURL:              "dummy_url",
				ManualMerge:           true,
				ShouldMergePR:         true,
			},
			patchSource: MockPatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutPRDiffFileMethod,
		},
		{
			name: "PR - no merge - no fork - auto merge - head branch",
			cfg: Config{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRMergeBranch: "pull/7/merge",
				PRHeadBranch:  "pull/7/head",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ManualMerge:   true,
				ShouldMergePR: false,
			},
			want: CheckoutHeadBranchCommitMethod,
		},
		{
			name: "PR - no merge - no fork - manual merge",
			cfg: Config{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ManualMerge:   true,
				ShouldMergePR: false,
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "PR - no merge - no fork - diff file exists",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
				Commit:        "76a934ae",
				PRDestBranch:  "master",
				PRID:          7,
				ShouldMergePR: false,
				BuildURL:      "dummy_url",
			},
			patchSource: MockPatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutCommitMethod,
		},
		{
			name: "PR - no merge - fork - public fork",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "https://github.com/bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         false,
			},
			want: CheckoutForkCommitMethod,
		},
		{
			name: "PR - no merge - fork - auto merge - diff file: private fork",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				PRID:                  7,
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         false,
				BuildURL:              "dummy_url",
			},
			patchSource: MockPatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutPRDiffFileMethod,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := selectCheckoutMethod(tt.cfg, tt.patchSource); got != tt.want {
				t.Errorf("selectCheckoutMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_commitInfoRef(t *testing.T) {
	tests := []struct {
		name     string
		strategy checkoutStrategy
		wantRef  string
	}{
		{
			strategy: checkoutNone{},
			wantRef:  "",
		},
		{
			strategy: checkoutCommit{
				params: CommitParams{
					Commit: "abcdef",
				},
			},
			wantRef: "abcdef",
		},
		{
			strategy: checkoutBranch{
				params: BranchParams{
					Branch: "hcnarb",
				},
			},
			wantRef: "refs/heads/hcnarb",
		},
		{
			strategy: checkoutTag{
				params: TagParams{
					Tag: "gat",
				},
			},
			wantRef: "refs/tags/gat",
		},
		{
			name:     "Does not support commit info",
			strategy: checkoutPRDiffFile{},
			wantRef:  "",
		},
		{
			name: "Non-fork",
			strategy: checkoutPRManualMerge{
				params: PRManualMergeParams{
					SourceBranch:      "source",
					SourceMergeArg:    "abcdef",
					DestinationBranch: "destBranch",
				},
			},
			wantRef: "abcdef",
		},
		{
			name: "Fork",
			strategy: checkoutPRManualMerge{
				params: PRManualMergeParams{
					SourceBranch:      "source",
					SourceMergeArg:    "remote/source",
					DestinationBranch: "destbranch",
					SourceRepoURL:     ".",
				},
			},
			wantRef: "remote/source",
		},
		{
			strategy: checkoutPRMergeBranch{
				params: PRMergeBranchParams{
					DestinationBranch: "dest",
					MergeBranch:       "pull/2/merge",
				},
			},
			wantRef: "pull/2",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T %s", tt.strategy, tt.name), func(t *testing.T) {
			gotRef := tt.strategy.commitInfoRef()

			assert.Equal(t, tt.wantRef, gotRef)
		})
	}
}
