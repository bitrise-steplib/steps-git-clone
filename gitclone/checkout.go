package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
)

type checkoutStrategy interface {
	Validate() error
	Do(gitCmd git.Git, fetchOptions fetchOptions) error
}

func selectCheckoutStrategy(cfg Config) (checkoutStrategy, fetchOptions, error) {
	defaultFetchTraits := fetchOptions{
		depth: cfg.CloneDepth,
		tags:  cfg.Tag != "",
	}

	isPR := cfg.PRRepositoryURL != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if !isPR {
		if cfg.Commit != "" {
			return checkoutCommit{
					commit: cfg.Commit,
				},
				defaultFetchTraits,
				nil
		}

		if cfg.Tag != "" {
			var branch *string
			if cfg.Branch != "" {
				branch = &cfg.Branch
			}

			return checkoutTag{
					tag:    cfg.Tag,
					branch: branch,
				},
				fetchOptions{
					depth: cfg.CloneDepth,
					tags:  true,
				}, nil
		}

		if cfg.Branch != "" {
			return checkoutBranch{
					branch: cfg.Branch,
				},
				defaultFetchTraits,
				nil
		}

		return checkoutNone{}, fetchOptions{}, nil
	}

	// ** PR **
	isPrivateFork := isPrivate(cfg.PRRepositoryURL) && isFork(cfg.RepositoryURL, cfg.PRRepositoryURL)
	if !cfg.ManualMerge || isPrivateFork { // Auto merge
		// Merge branch
		if cfg.PRMergeBranch != "" {
			return checkoutPullRequestAutoMergeBranch{
					baseBranch:  cfg.BranchDest,
					mergeBranch: cfg.PRMergeBranch,
				},
				fetchOptions{
					depth: cfg.CloneDepth,
					tags:  false,
				},
				nil
		}

		// Diff file
		if patch, err := getDiffFile(cfg.BuildURL, cfg.BuildAPIToken, cfg.PRID); err != nil {
			return nil, fetchOptions{}, fmt.Errorf("merging PR (automatic) failed, there is no Pull Request branch and can't download diff file: %v", err)
		} else {
			return checkoutPullRequestAutoDiffFile{
					baseBranch: cfg.BranchDest,
					patch:      patch,
				},
				fetchOptions{
					depth: cfg.CloneDepth,
					tags:  false,
				}, nil
		}
	}

	// ** PR/MR with manual merge
	// Clone Depth is not set for manual merge yet
	if isFork(cfg.RepositoryURL, cfg.PRRepositoryURL) {
		return checkoutForkPullRequestManual{
				branchFork:  cfg.Branch,
				forkRepoURL: cfg.PRRepositoryURL,
				branchBase:  cfg.BranchDest,
			},
			fetchOptions{},
			nil
	}

	return checkoutMergeRequestManual{
			branchHead: cfg.Branch,
			branchBase: cfg.BranchDest,
			commit:     cfg.Commit,
		},
		fetchOptions{},
		nil
}
