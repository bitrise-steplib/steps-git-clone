package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

// PRMergeBranchParams are parameters to check out a Merge/Pull Request (when a merge branch is available)
type PRMergeBranchParams struct {
	DestinationBranch string
	// Merge branch contains the changes premerged by the Git provider
	MergeBranch string
}

// NewPRMergeBranchParams validates and returns a new PRMergeBranchParams
func NewPRMergeBranchParams(destBranch, mergeBranch string) (*PRMergeBranchParams, error) {
	if strings.TrimSpace(destBranch) == "" {
		return nil, NewParameterValidationError("PR merge branch based checkout strategy can not be used: no destination branch specified")
	}
	if strings.TrimSpace(mergeBranch) == "" {
		return nil, NewParameterValidationError("PR merge branch based checkout strategy can not be used: no merge branch specified")
	}

	return &PRMergeBranchParams{
		DestinationBranch: destBranch,
		MergeBranch:       mergeBranch,
	}, nil
}

// checkoutPRMergeBranch
type checkoutPRMergeBranch struct {
	params PRMergeBranchParams
}

func (c checkoutPRMergeBranch) do(gitCmd git.Git, fetchOpts fetchOptions, fallback fallbackRetry) error {
	// Check out initial branch (fetchInitialBranch part1)
	// `git "fetch" "origin" "refs/heads/master"`
	destBranchRef := refsHeadsPrefix + c.params.DestinationBranch
	if err := fetch(gitCmd, originRemoteName, destBranchRef, fetchOpts); err != nil {
		return err
	}

	// `git "fetch" "origin" "refs/pull/7/head:pull/7"`
	remoteRef, localRef := headBranchRefs(c.params.MergeBranch)
	if err := fetch(gitCmd, originRemoteName, fmt.Sprintf("%s:%s", remoteRef, localRef), fetchOpts); err != nil {
		return err
	}

	// Check out initial branch (fetchInitialBranch part2)
	// `git "checkout" "master"`
	// `git "merge" "origin/master"`
	if err := checkoutWithCustomRetry(gitCmd, c.params.DestinationBranch, nil); err != nil {
		return err
	}
	destBranchWithRemote := fmt.Sprintf("%s/%s", originRemoteName, c.params.DestinationBranch)
	if err := runner.Run(gitCmd.Merge(destBranchWithRemote)); err != nil {
		return err
	}

	// `git "merge" "pull/7"`
	if err := mergeWithCustomRetry(gitCmd, localRef, fallback); err != nil {
		return err
	}

	return detachHead(gitCmd)
}

func (c checkoutPRMergeBranch) getBuildTriggerRef() string {
	_, localRef := headBranchRefs(c.params.MergeBranch)

	return localRef
}
