package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

// CheckoutForkBranchParams are parameters to check out a PR branch from a fork, without merging
type CheckoutForkBranchParams struct {
	SourceRepoURL, SourceBranch string
}

// NewCheckoutForkBranchParams validates and returns a new CheckoutForkBranchParams
func NewCheckoutForkBranchParams(sourceBranch, sourceRepoURL string) (*CheckoutForkBranchParams, error) {
	if strings.TrimSpace(sourceRepoURL) == "" {
		return nil, NewParameterValidationError("PR (fork) source branch checkout strategy can not be used: no source repository URL specified")
	}
	if strings.TrimSpace(sourceBranch) == "" {
		return nil, NewParameterValidationError("PR (fork) source branch checkout strategy can not be used: no source branch specified")
	}

	return &CheckoutForkBranchParams{
		SourceRepoURL: sourceRepoURL,
		SourceBranch:  sourceBranch,
	}, nil
}

type checkoutForkBranch struct {
	params CheckoutForkBranchParams
}

func (c checkoutForkBranch) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.SourceRepoURL)); err != nil {
		return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.SourceRepoURL, err)
	}

	sourceBranchRef := refsHeadsPrefix + c.params.SourceBranch
	if err := fetchInitialBranch(gitCmd, forkRemoteName, sourceBranchRef, fetchOptions); err != nil {
		return err
	}

	return nil
}

// CheckoutHeadBranchParams are parameters to check out a head branch (provided by the git hosting service)
type CheckoutHeadBranchParams struct {
	HeadBranch string
	Commit     string
}

// NewCheckoutHeadBranchParams validates and returns a new NewCheckoutHeadBranchParams
func NewCheckoutHeadBranchParams(specialHeadBranch, commit string) (*CheckoutHeadBranchParams, error) {
	if strings.TrimSpace(specialHeadBranch) == "" {
		return nil, NewParameterValidationError("PR head branch checkout strategy can not be used: no head branch specified")
	}
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("PR head branch checkout stategy can not be used: no commit specified")
	}

	return &CheckoutHeadBranchParams{
		HeadBranch: specialHeadBranch,
	}, nil
}

type checkoutHeadBranch struct {
	params CheckoutHeadBranchParams
}

func (c checkoutHeadBranch) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	branchRef := refsPrefix + c.params.HeadBranch                                  // ref/pull/2/head
	trackingBranch := c.params.HeadBranch                                          // pull/2/head
	branchRefWithTrackingBranch := fmt.Sprintf("%s:%s", branchRef, trackingBranch) // ref/pull/2/head:pull/2/head
	if err := fetch(gitCmd, originRemoteName, branchRefWithTrackingBranch, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, trackingBranch, nil); err != nil {
		return err
	}

	return nil
}
