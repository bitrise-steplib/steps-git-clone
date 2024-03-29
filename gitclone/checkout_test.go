package gitclone

import (
	"fmt"
	"testing"

	"github.com/bitrise-steplib/steps-git-clone/gitclone/bitriseapi"
	"github.com/stretchr/testify/assert"
)

func Test_selectCheckoutMethod(t *testing.T) {
	tests := []struct {
		name            string
		cfg             Config
		patchSource     bitriseapi.PatchSource
		mergeRefChecker bitriseapi.MergeRefChecker
		want            CheckoutMethod
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
			name: "PR - no fork - branch and commit",
			cfg: Config{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRMergeRef:    "pull/7/merge",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ShouldMergePR: true,
			},
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - unverified merge ref - is up to date",
			cfg: Config{
				Commit:               "76a934ae",
				Branch:               "test/commit-messages",
				PRUnverifiedMergeRef: "pull/7/merge",
				PRDestBranch:         "master",
				CloneDepth:           1,
				ShouldMergePR:        true,
			},
			mergeRefChecker: FakeMergeRefChecker{isUpToDate: true},
			want:            CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - unverified merge ref - does not become up to date",
			cfg: Config{
				Commit:               "76a934ae",
				Branch:               "test/commit-messages",
				PRUnverifiedMergeRef: "pull/7/merge",
				PRDestBranch:         "master",
				CloneDepth:           1,
				ShouldMergePR:        true,
			},
			mergeRefChecker: FakeMergeRefChecker{isUpToDate: false},
			patchSource:     FakePatchSource{err: fmt.Errorf("no patch file")},
			want:            CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - unverified merge ref - error",
			cfg: Config{
				Commit:               "76a934ae",
				Branch:               "test/commit-messages",
				PRUnverifiedMergeRef: "pull/7/merge",
				PRDestBranch:         "master",
				CloneDepth:           1,
				ShouldMergePR:        true,
			},
			mergeRefChecker: FakeMergeRefChecker{err: fmt.Errorf("error while checking merge ref")},
			patchSource:     FakePatchSource{err: fmt.Errorf("no patch file")},
			want:            CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - branch and commit, no PRRepoURL or PRID",
			cfg: Config{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ShouldMergePR: true,
			},
			patchSource: FakePatchSource{err: fmt.Errorf("no patch file available")},
			want:        CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - fork - no merge ref - no patch file",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "https://github.com/bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ShouldMergePR:         true,
			},
			patchSource: FakePatchSource{err: fmt.Errorf("no patch file available")},
			want:        CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - manual merge: repo is the same with different scheme",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/git-clone-test.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				PRMergeRef:            "pull/7/merge",
				Commit:                "76a934ae",
				ShouldMergePR:         true,
			},
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - no fork - merge ref (GitHub format)",
			cfg: Config{
				PRDestBranch:  "master",
				PRMergeRef:    "pull/5/merge",
				ShouldMergePR: true,
			},
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - no fork - diff file",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
				PRDestBranch:  "master",
				Commit:        "76a934ae",
				ShouldMergePR: true,
			},
			patchSource: FakePatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutPRDiffFileMethod,
		},
		{
			name: "PR - no merge - no fork - head branch",
			cfg: Config{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRMergeRef:    "pull/7/merge",
				PRHeadBranch:  "pull/7/head",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ShouldMergePR: false,
			},
			want: CheckoutHeadBranchCommitMethod,
		},
		{
			name: "PR - no merge - no fork - no PR head - no merge ref",
			cfg: Config{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRDestBranch:  "master",
				CloneDepth:    1,
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
				ShouldMergePR: false,
			},
			patchSource: FakePatchSource{diffFilePath: "dummy_path"},
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
				ShouldMergePR:         false,
			},
			want: CheckoutForkCommitMethod,
		},
		{
			name: "PR - no merge - fork - diff file: private fork",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ShouldMergePR:         false,
			},
			patchSource: FakePatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutPRDiffFileMethod,
		},
		{
			name: "PR - no merge - fork - diff file doesn't exist",
			cfg: Config{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ShouldMergePR:         false,
			},
			patchSource: FakePatchSource{diffFilePath: ""},
			want:        CheckoutForkCommitMethod,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := selectCheckoutMethod(tt.cfg, tt.patchSource, tt.mergeRefChecker); got != tt.want {
				t.Errorf("selectCheckoutMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBuildTriggerRef(t *testing.T) {
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
			strategy: checkoutPRMergeRef{
				params: PRMergeRefParams{
					MergeRef: "pull/2/merge",
					HeadRef:  "pull/2/head",
				},
			},
			wantRef: "refs/remotes/pull/2/head",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T %s", tt.strategy, tt.name), func(t *testing.T) {
			gotRef := tt.strategy.getBuildTriggerRef()

			assert.Equal(t, tt.wantRef, gotRef)
		})
	}
}

func Test_idealDefaultCloneDepth(t *testing.T) {
	tests := []struct {
		method CheckoutMethod
		want   int
	}{
		{
			method: CheckoutNoneMethod,
			want:   1,
		},
		{
			method: CheckoutPRMergeBranchMethod,
			want:   1,
		},
		{
			method: CheckoutPRManualMergeMethod,
			want:   50,
		},
		{
			method: CheckoutPRDiffFileMethod,
			want:   1,
		},
		{
			method: CheckoutCommitMethod,
			want:   1,
		},
		{
			method: CheckoutTagMethod,
			want:   1,
		},
		{
			method: CheckoutBranchMethod,
			want:   1,
		},
		{
			method: CheckoutHeadBranchCommitMethod,
			want:   1,
		},
		{
			method: CheckoutForkCommitMethod,
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.method.String(), func(t *testing.T) {
			if got := idealDefaultCloneDepth(tt.method); got != tt.want {
				t.Errorf("idealDefaultCloneDepth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_selectFetchOptions(t *testing.T) {
	type args struct {
		method          CheckoutMethod
		cloneDepth      int
		fetchTags       bool
		fetchSubmodules bool
		filterTree      bool
	}
	tests := []struct {
		name string
		args args
		want fetchOptions
	}{
		{
			name: "default depth setting",
			args: args{
				method:          CheckoutCommitMethod,
				cloneDepth:      0,
				fetchTags:       false,
				fetchSubmodules: false,
				filterTree:      false,
			},
			want: fetchOptions{
				tags:            false,
				limitDepth:      true,
				depth:           1,
				fetchSubmodules: false,
				filterTree:      false,
			},
		},
		{
			name: "custom depth setting",
			args: args{
				method:          CheckoutPRMergeBranchMethod,
				cloneDepth:      115,
				fetchTags:       false,
				fetchSubmodules: false,
				filterTree:      false,
			},
			want: fetchOptions{
				tags:            false,
				limitDepth:      true,
				depth:           115,
				fetchSubmodules: false,
				filterTree:      false,
			},
		},
		{
			name: "disable depth limit",
			args: args{
				method:          CheckoutCommitMethod,
				cloneDepth:      -1,
				fetchTags:       false,
				fetchSubmodules: false,
				filterTree:      false,
			},
			want: fetchOptions{
				tags:            false,
				limitDepth:      false,
				depth:           -1,
				fetchSubmodules: false,
				filterTree:      false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, selectFetchOptions(tt.args.method, tt.args.cloneDepth, tt.args.fetchTags, tt.args.fetchSubmodules, tt.args.filterTree), "selectFetchOptions(%v, %v, %v, %v, %v)", tt.args.method, tt.args.cloneDepth, tt.args.fetchTags, tt.args.fetchSubmodules, tt.args.filterTree)
		})
	}
}
