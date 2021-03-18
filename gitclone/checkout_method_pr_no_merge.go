package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

// CheckoutForkCommitParams are parameters to check out a PR branch from a fork, without merging
type CheckoutForkCommitParams struct {
	SourceRepoURL, SourceBranch, Commit string
}

// NewCheckoutForkCommitParams validates and returns a new CheckoutForkBranchParams
func NewCheckoutForkCommitParams(sourceBranch, sourceRepoURL, commit string) (*CheckoutForkCommitParams, error) {
	if strings.TrimSpace(sourceRepoURL) == "" {
		return nil, NewParameterValidationError("PR (fork) commit checkout strategy can not be used: no source repository URL specified")
	}
	if strings.TrimSpace(sourceBranch) == "" {
		return nil, NewParameterValidationError("PR (fork) commit checkout strategy can not be used: no source branch specified")
	}
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("PR (fork) commit checkout strategy can not be used: no commit specified")
	}

	return &CheckoutForkCommitParams{
		SourceRepoURL: sourceRepoURL,
		SourceBranch:  sourceBranch,
		Commit:        commit,
	}, nil
}

type checkoutForkCommit struct {
	params CheckoutForkCommitParams
}

func (c checkoutForkCommit) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.SourceRepoURL)); err != nil {
		return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.SourceRepoURL, err)
	}

	sourceBranchRef := refsHeadsPrefix + c.params.SourceBranch
	if err := fetch(gitCmd, forkRemoteName, sourceBranchRef, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.Commit, fallback); err != nil {
		return err
	}

	return nil
}

// CheckoutHeadBranchCommitParams are parameters to check out a head branch (provided by the git hosting service)
type CheckoutHeadBranchCommitParams struct {
	HeadBranch string
	Commit     string
}

// NewCheckoutHeadBranchCommitParams validates and returns a new NewCheckoutHeadBranchCommitParams
func NewCheckoutHeadBranchCommitParams(specialHeadBranch, commit string) (*CheckoutHeadBranchCommitParams, error) {
	if strings.TrimSpace(specialHeadBranch) == "" {
		return nil, NewParameterValidationError("PR head branch checkout strategy can not be used: no head branch specified")
	}
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("PR head branch checkout stategy can not be used: no commit specified")
	}

	return &CheckoutHeadBranchCommitParams{
		HeadBranch: specialHeadBranch,
		Commit:     commit,
	}, nil
}

type checkoutHeadBranchCommit struct {
	params CheckoutHeadBranchCommitParams
}

func (c checkoutHeadBranchCommit) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	branchRef := refsPrefix + c.params.HeadBranch // ref/pull/2/head
	if err := fetch(gitCmd, originRemoteName, branchRef, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.Commit, fallback); err != nil {
		return err
	}

	return nil
}
