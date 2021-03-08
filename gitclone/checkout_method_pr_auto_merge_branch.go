package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

// PRMergeBranchParams are parameters to check out a Merge/Pull Request if merge branch is available
type PRMergeBranchParams struct {
	BaseBranch string
	// Merge branch contains the changes premerged by the Git provider
	MergeBranch string
}

// NewPRMergeBranchParams validates and returns a new PRMergeBranchParams
func NewPRMergeBranchParams(baseBranch, mergeBranch string) (*PRMergeBranchParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR merge branch based checkout strategy can not be used, no base branch specified")
	}
	if strings.TrimSpace(mergeBranch) == "" {
		return nil, NewParameterValidationError("PR merge branch based checkout strategy can not be used, no merge branch specified")
	}

	return &PRMergeBranchParams{
		BaseBranch:  baseBranch,
		MergeBranch: mergeBranch,
	}, nil
}

//
// checkoutPRMergeBranch
type checkoutPRMergeBranch struct {
	params PRMergeBranchParams
}

func (c checkoutPRMergeBranch) do(gitCmd git.Git, fetchOpts fetchOptions) error {
	// Check out initial branch (fetchInitialBranch part1)
	// `git "fetch" "origin" "refs/heads/master"`
	baseBranchRef := newOriginFetchRef(branchRefPrefix + c.params.BaseBranch)
	if err := fetch(gitCmd, fetchOpts, baseBranchRef); err != nil {
		return err
	}

	// `git "fetch" "origin" "refs/pull/7/head:pull/7"`
	// Does not apply clone depth (legacy)
	headBranchRef := newOriginFetchRef(fetchArg(c.params.MergeBranch))
	if err := fetch(gitCmd, fetchOptions{}, headBranchRef); err != nil {
		return err
	}

	// Check out initial branch (fetchInitialBranch part2)
	// `git "checkout" "master"`
	// `git "merge" "origin/master"`
	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{arg: c.params.BaseBranch, isBranch: true}, nil); err != nil {
		return err
	}
	remoteBaseBranch := fmt.Sprintf("%s/%s", defaultRemoteName, c.params.BaseBranch)
	if err := runner.Run(gitCmd.Merge(remoteBaseBranch)); err != nil {
		return err
	}

	// `git "merge" "pull/7"`
	var resetFunc func(git.Git, error) error
	if !fetchOpts.IsFullDepth() {
		resetFunc = func(gitCmd git.Git, merr error) error {
			log.Warnf("Merge failed: %v\nReset repository, then unshallow...", merr)

			if err := resetRepo(gitCmd); err != nil {
				return fmt.Errorf("reset repository: %v", err)
			}
			if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
				return fmt.Errorf("fetch failed: %v", err)
			}

			return nil
		}
	}
	if err := mergeWithCustomRetry(gitCmd, mergeArg(c.params.MergeBranch), resetFunc); err != nil {
		return err
	}

	return detachHead(gitCmd)
}
