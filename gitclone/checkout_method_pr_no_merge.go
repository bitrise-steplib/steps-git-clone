package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

type CheckoutNoMergeForkBranchParams struct {
	HeadRepoURL, HeadBranch string
}

func NewCheckoutNoMergeForkBranchParams(headBranch, forkRepoURL string) (*CheckoutNoMergeForkBranchParams, error) {
	if strings.TrimSpace(forkRepoURL) == "" {
		return nil, NewParameterValidationError("PR (fork) head branch checkout strategy can not be used: no head repository URL specified")
	}
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewParameterValidationError("PR (fork) head branch checkout strategy can not be used: no head branch specified")
	}

	return &CheckoutNoMergeForkBranchParams{
		HeadRepoURL: forkRepoURL,
		HeadBranch:  headBranch,
	}, nil
}

type checkoutForkBranch struct {
	params CheckoutNoMergeForkBranchParams
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

type CheckoutNoMergeSpecialHeadBranchParams struct {
	SpecialHeadBranch string
}

func NewCheckoutNoMergeSpecialHeadBranchParams(specialHeadBranch string) (*CheckoutNoMergeSpecialHeadBranchParams, error) {
	if strings.TrimSpace(specialHeadBranch) == "" {
		return nil, NewParameterValidationError("PR special head branch checkout strategy can not be used: no head branch specified")
	}

	return &CheckoutNoMergeSpecialHeadBranchParams{
		SpecialHeadBranch: specialHeadBranch,
	}, nil
}

type checkoutSpecialHeadBranch struct {
	params CheckoutNoMergeSpecialHeadBranchParams
}

func (c checkoutSpecialHeadBranch) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	branchRef := refsPrefix + c.params.SpecialHeadBranch
	trackingBranch := c.params.SpecialHeadBranch
	if err := fetch(gitCmd, originRemoteName, fmt.Sprintf("%s:%s", branchRef, trackingBranch), fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, trackingBranch, nil); err != nil {
		return err
	}

	return nil
}
