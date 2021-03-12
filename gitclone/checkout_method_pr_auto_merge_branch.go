package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

// PRMergeBranchParams are parameters to check out a Merge/Pull Request (when a merge branch is available)
type PRMergeBranchParams struct {
	BaseBranch string
	// Merge branch contains the changes premerged by the Git provider
	MergeBranch string
}

// NewPRMergeBranchParams validates and returns a new PRMergeBranchParams
func NewPRMergeBranchParams(baseBranch, mergeBranch string) (*PRMergeBranchParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR merge branch based checkout strategy can not be used: no base branch specified")
	}
	if strings.TrimSpace(mergeBranch) == "" {
		return nil, NewParameterValidationError("PR merge branch based checkout strategy can not be used: no merge branch specified")
	}

	return &PRMergeBranchParams{
		BaseBranch:  baseBranch,
		MergeBranch: mergeBranch,
	}, nil
}

// checkoutPRMergeBranch
type checkoutPRMergeBranch struct {
	params PRMergeBranchParams
}

func (c checkoutPRMergeBranch) do(gitCmd git.Git, fetchOpts fetchOptions, fallback fallbackRetry) error {
	// Check out initial branch (fetchInitialBranch part1)
	// `git "fetch" "origin" "refs/heads/master"`
	baseBranchRef := branchRefPrefix + c.params.BaseBranch
	if err := fetch(gitCmd, originRemoteName, &baseBranchRef, fetchOpts); err != nil {
		return err
	}

	// `git "fetch" "origin" "refs/pull/7/head:pull/7"`
	headBranchRef := fetchArg(c.params.MergeBranch)
	if err := fetch(gitCmd, originRemoteName, &headBranchRef, fetchOpts); err != nil {
		return err
	}

	// Check out initial branch (fetchInitialBranch part2)
	// `git "checkout" "master"`
	// `git "merge" "origin/master"`
	if err := checkoutWithCustomRetry(gitCmd, c.params.BaseBranch, nil); err != nil {
		return err
	}
	remoteBaseBranch := fmt.Sprintf("%s/%s", originRemoteName, c.params.BaseBranch)
	if err := runner.Run(gitCmd.Merge(remoteBaseBranch)); err != nil {
		return err
	}

	// `git "merge" "pull/7"`
	if err := mergeWithCustomRetry(gitCmd, mergeArg(c.params.MergeBranch), fallback); err != nil {
		return err
	}

	return detachHead(gitCmd)
}
