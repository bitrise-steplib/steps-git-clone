package gitclone

import "testing"

func Test_selectCheckoutMethod(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want CheckoutMethod
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
				BranchDest:    "master",
				PRID:          7,
				CloneDepth:    1,
				ManualMerge:   true,
			},
			want: CheckoutPRManualMergeMethod,
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
			want: CheckoutCommitMethod,
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
			want: CheckoutPRManualMergeMethod,
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
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - auto merge - merge branch (GitHub format)",
			cfg: Config{
				BranchDest:    "master",
				PRMergeBranch: "pull/5/merge",
			},
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - fork - auto merge - diff file",
			cfg: Config{
				RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
				BranchDest:    "master",
				PRID:          7,
				Commit:        "76a934ae",
			},
			want: CheckoutPRDiffFileMethod,
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
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - fork - auto merge: private fork overrides manual merge flag, Fails",
			cfg: Config{
				RepositoryURL:   "https://github.com/bitrise-io/git-clone-test.git",
				PRRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:          "test/commit-messages",
				BranchDest:      "master",
				Commit:          "76a934ae",
				ManualMerge:     true,
			},
			want: CheckoutPRDiffFileMethod,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectCheckoutMethod(tt.cfg); got != tt.want {
				t.Errorf("selectCheckoutMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}
