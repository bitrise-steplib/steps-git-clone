package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
)

type checkoutStrategy interface {
	Validate() error
	Do(gitCmd git.Git) error
}

func selectCheckoutStrategy(cfg Config) (checkoutStrategy, error) {
	defaultFetchTraits := fetchTraits{
		Depth: cfg.CloneDepth,
		Tags:  cfg.Tag != "",
	}

	isPR := cfg.PRRepositoryURL != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if !isPR {
		if cfg.Commit != "" {
			return checkoutCommit{
				Commit:      cfg.Commit,
				FetchTraits: defaultFetchTraits,
			}, nil
		}

		if cfg.Tag != "" {
			var branch *string
			if cfg.Branch != "" {
				branch = &cfg.Branch
			}

			return checkoutTag{
				Tag:    cfg.Tag,
				Branch: branch,
				FetchTraits: fetchTraits{
					Depth: cfg.CloneDepth,
					Tags:  true,
				},
			}, nil
		}

		if cfg.Branch != "" {
			return checkoutBranch{
				Branch:      cfg.Branch,
				FetchTraits: defaultFetchTraits,
			}, nil
		}

		return checkoutNone{}, nil
	}

	// ** PR **
	isPrivateFork := isPrivate(cfg.PRRepositoryURL) && isFork(cfg.RepositoryURL, cfg.PRRepositoryURL)
	if !cfg.ManualMerge || isPrivateFork { // Auto merge
		// Merge branch
		if cfg.PRMergeBranch != "" {
			return checkoutPullRequestAutoMergeBranch{
				baseBranch:  cfg.BranchDest,
				mergeBranch: cfg.PRMergeBranch,
				fetchTraits: fetchTraits{
					Depth: cfg.CloneDepth,
					Tags:  false,
				},
			}, nil
		}

		// Diff file
		if patch, err := getDiffFile(cfg.BuildURL, cfg.BuildAPIToken, cfg.PRID); err != nil {
			return nil, fmt.Errorf("merging PR (automatic) failed, there is no Pull Request branch and can't download diff file: %v", err)
		} else {
			return checkoutPullRequestAutoDiffFile{
				baseBranch: cfg.BranchDest,
				patch:      patch,
				fetchTraits: fetchTraits{
					Depth: cfg.CloneDepth,
					Tags:  false,
				},
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
		}, nil
	}

	return checkoutMergeRequestManual{
		branchHead: cfg.Branch,
		branchBase: cfg.BranchDest,
		commit:     cfg.Commit,
	}, nil
}
