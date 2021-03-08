package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
)

// CheckoutMethod is the checkout method used
type CheckoutMethod int

const (
	// InvalidCheckoutMethod ...
	InvalidCheckoutMethod CheckoutMethod = iota
	// CheckoutNoneMethod only adds remote, resets repo, updates submodules
	CheckoutNoneMethod
	// CheckoutCommitMethod checks out a commit
	CheckoutCommitMethod
	// CheckoutTagMethod checks out a tag
	CheckoutTagMethod
	// CheckoutBranchMethod checks out a branch
	CheckoutBranchMethod
	// CheckoutPRMergeBranchMethod checks out a MR/PR in case a merge branch is available
	CheckoutPRMergeBranchMethod
	// CheckoutPRDiffFileMethod  checks out a MR/PR in case a diff file is available
	CheckoutPRDiffFileMethod
	// CheckoutPRManualMergeMethod checks out a MR
	CheckoutPRManualMergeMethod
	// CheckoutForkPRManualMergeMethod checks out a PR
	CheckoutForkPRManualMergeMethod
)

// ParameterValidationError is returned when there is missing or malformatted parameter for a given parameter set
type ParameterValidationError struct {
	ErrorString string
}

// Error ...
func (e ParameterValidationError) Error() string {
	return e.ErrorString
}

// NewParameterValidationError return a new ValidationError
func NewParameterValidationError(msg string) error {
	return ParameterValidationError{ErrorString: msg}
}

// checkoutStrategy is the interface an actual checkout strategy implements
type checkoutStrategy interface {
	do(gitCmd git.Git, fetchOptions fetchOptions) error
}

func selectCheckoutMethod(cfg Config) CheckoutMethod {
	isPR := cfg.PRRepositoryURL != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if !isPR {
		if cfg.Commit != "" {
			return CheckoutCommitMethod
		}

		if cfg.Tag != "" {
			return CheckoutTagMethod
		}

		if cfg.Branch != "" {
			return CheckoutBranchMethod
		}

		return CheckoutNoneMethod
	}

	// ** PR **
	isPrivateFork := isPrivate(cfg.PRRepositoryURL) && isFork(cfg.RepositoryURL, cfg.PRRepositoryURL)
	if !cfg.ManualMerge || isPrivateFork { // Auto merge
		// Merge branch
		if cfg.PRMergeBranch != "" {
			return CheckoutPRMergeBranchMethod
		}

		return CheckoutPRDiffFileMethod
	}

	// ** PR/MR with manual merge
	if isFork(cfg.RepositoryURL, cfg.PRRepositoryURL) {
		return CheckoutForkPRManualMergeMethod
	}

	return CheckoutPRManualMergeMethod
}

func createCheckoutStrategy(checkoutMethod CheckoutMethod, cfg Config, patch string) (checkoutStrategy, error) {
	switch checkoutMethod {
	case CheckoutNoneMethod:
		{
			return checkoutNone{}, nil
		}
	case CheckoutCommitMethod:
		{
			params, err := NewCommitParams(cfg.Commit)
			if err != nil {
				return nil, err
			}

			return checkoutCommit{
				params: *params,
			}, nil
		}
	case CheckoutTagMethod:
		{
			var branch *string
			if cfg.Branch != "" {
				branch = &cfg.Branch
			}
			params, err := NewTagParams(cfg.Tag, branch)
			if err != nil {
				return nil, err
			}

			return checkoutTag{
				params: *params,
			}, nil
		}
	case CheckoutBranchMethod:
		{
			params, err := NewBranchParams(cfg.Branch, nil)
			if err != nil {
				return nil, err
			}

			return checkoutBranch{
				params: *params,
			}, nil
		}
	case CheckoutPRMergeBranchMethod:
		{
			params, err := NewPRMergeBranchParams(cfg.BranchDest, cfg.PRMergeBranch)
			if err != nil {
				return nil, err
			}

			return checkoutPRMergeBranch{
				params: *params,
			}, nil
		}
	case CheckoutPRDiffFileMethod:
		{
			return checkoutPRDiffFile{
				baseBranch: cfg.BranchDest,
				patch:      patch,
			}, nil
		}
	case CheckoutForkPRManualMergeMethod:
		{
			params, err := NewForkPRManualMergeParams(cfg.Branch, cfg.PRRepositoryURL, cfg.BranchDest)
			if err != nil {
				return nil, err
			}

			return checkoutForkPRManualMerge{
				params: *params,
			}, nil
		}
	case CheckoutPRManualMergeMethod:
		{
			params, err := NewPRManualMergeParams(cfg.Branch, cfg.Commit, cfg.BranchDest)
			if err != nil {
				return nil, err
			}

			return checkoutPRManualMerge{
				params: *params,
			}, nil
		}
	default:
		return nil, fmt.Errorf("invalid checkout strategy selected")
	}

}

func selectfetchOptions(checkoutStrategy CheckoutMethod, cloneDepth int, isTag bool) fetchOptions {
	defaultFetchTraits := fetchOptions{
		depth: cloneDepth,
		tags:  isTag,
	}

	switch checkoutStrategy {
	case CheckoutCommitMethod, CheckoutBranchMethod:
		return defaultFetchTraits
	case CheckoutTagMethod:
		return fetchOptions{
			depth: cloneDepth,
			tags:  true, // Needed to check out tags
		}
	case CheckoutPRMergeBranchMethod, CheckoutPRDiffFileMethod:
		return fetchOptions{
			depth: cloneDepth,
			tags:  false,
		}
	// Clone Depth is not set for manual merge yet
	case CheckoutPRManualMergeMethod, CheckoutForkPRManualMergeMethod, CheckoutNoneMethod:
		return fetchOptions{}
	default:
		return fetchOptions{}
	}
}
