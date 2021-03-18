package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

// CheckoutForkBranchParams are parameters to check out a PR branch from a fork, without merging
type CheckoutForkBranchParams struct {
	HeadRepoURL, HeadBranch string
}

// NewCheckoutForkBranchParams validates and returns a new CheckoutForkBranchParams
func NewCheckoutForkBranchParams(headBranch, forkRepoURL string) (*CheckoutForkBranchParams, error) {
	if strings.TrimSpace(forkRepoURL) == "" {
		return nil, NewParameterValidationError("PR (fork) head branch checkout strategy can not be used: no head repository URL specified")
	}
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewParameterValidationError("PR (fork) head branch checkout strategy can not be used: no head branch specified")
	}

	return &CheckoutForkBranchParams{
		HeadRepoURL: forkRepoURL,
		HeadBranch:  headBranch,
	}, nil
}

type checkoutForkBranch struct {
	params CheckoutForkBranchParams
}

func (c checkoutForkBranch) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.HeadRepoURL)); err != nil {
		return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.HeadRepoURL, err)
	}

	forkBranchRef := refsHeadsPrefix + c.params.HeadBranch
	if err := fetchInitialBranch(gitCmd, forkRemoteName, forkBranchRef, fetchOptions); err != nil {
		return err
	}

	return nil
}

// CheckoutHeadBranchParams are parameters to check out a head branch (provided by the git hosting service)
type CheckoutHeadBranchParams struct {
	HeadBranch string
}

// NewCheckoutHeadBranchParams validates and returns a new NewCheckoutHeadBranchParams
func NewCheckoutHeadBranchParams(specialHeadBranch string) (*CheckoutHeadBranchParams, error) {
	if strings.TrimSpace(specialHeadBranch) == "" {
		return nil, NewParameterValidationError("PR special head branch checkout strategy can not be used: no head branch specified")
	}

	return &CheckoutHeadBranchParams{
		HeadBranch: specialHeadBranch,
	}, nil
}

type checkoutHeadBranch struct {
	params CheckoutHeadBranchParams
}

func (c checkoutHeadBranch) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	branchRef := refsPrefix + c.params.HeadBranch
	trackingBranch := c.params.HeadBranch
	if err := fetch(gitCmd, originRemoteName, fmt.Sprintf("%s:%s", branchRef, trackingBranch), fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, trackingBranch, nil); err != nil {
		return err
	}

	return nil
}
