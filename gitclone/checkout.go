package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/gitcloneparams"
)

type checkoutStrategy interface {
	do(gitCmd git.Git, fetchOptions fetchOptions) error
}

func selectCheckoutStrategy(cfg Config) (checkoutStrategy, fetchOptions, error) {
	defaultFetchTraits := fetchOptions{
		depth: cfg.CloneDepth,
		tags:  cfg.Tag != "",
	}

	isPR := cfg.PRRepositoryURL != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if !isPR {
		if cfg.Commit != "" {
			params, err := gitcloneparams.NewCommitParams(cfg.Commit)
			if err != nil {
				return nil, fetchOptions{}, err
			}

			return checkoutCommit{
					params: *params,
				},
				defaultFetchTraits,
				nil
		}

		if cfg.Tag != "" {
			var branch *string
			if cfg.Branch != "" {
				branch = &cfg.Branch
			}
			params, err := gitcloneparams.NewTagParams(cfg.Tag, branch)
			if err != nil {
				return nil, fetchOptions{}, err
			}

			return checkoutTag{
					params: *params,
				},
				fetchOptions{
					depth: cfg.CloneDepth,
					tags:  true,
				},
				nil
		}

		if cfg.Branch != "" {
			params, err := gitcloneparams.NewBranchParams(cfg.Branch, nil)
			if err != nil {
				return nil, fetchOptions{}, err
			}

			return checkoutBranch{
					params: *params,
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
			params, err := gitcloneparams.NewPRMergeBranchParams(cfg.BranchDest, cfg.PRMergeBranch)
			if err != nil {
				return nil, fetchOptions{}, err
			}

			return checkoutPRMergeBranch{
					params: *params,
				},
				fetchOptions{
					depth: cfg.CloneDepth,
					tags:  false,
				},
				nil
		}

		// Diff file
		patch, err := getDiffFile(cfg.BuildURL, cfg.BuildAPIToken, cfg.PRID)
		if err != nil {
			return nil, fetchOptions{}, fmt.Errorf("merging PR (automatic) failed, there is no Pull Request branch and can't download diff file: %v", err)
		}

		return checkoutPRDiffFile{
				baseBranch: cfg.BranchDest,
				patch:      patch,
			},
			fetchOptions{
				depth: cfg.CloneDepth,
				tags:  false,
			},
			nil
	}

	// ** PR/MR with manual merge
	// Clone Depth is not set for manual merge yet
	if isFork(cfg.RepositoryURL, cfg.PRRepositoryURL) {
		params, err := gitcloneparams.NewForkPRManualMergeParams(cfg.Branch, cfg.PRRepositoryURL, cfg.BranchDest)
		if err != nil {
			return nil, fetchOptions{}, err
		}

		return checkoutForkPRManualMerge{
				params: *params,
			},
			fetchOptions{},
			nil

	}

	params, err := gitcloneparams.NewPRManualMergeParams(cfg.Branch, cfg.Commit, cfg.BranchDest)
	if err != nil {
		return nil, fetchOptions{}, err
	}

	return checkoutPRManualMerge{
			params: *params,
		},
		fetchOptions{},
		nil
}
