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
	// CheckoutCommitMethod checks out a given commit
	CheckoutCommitMethod
	// CheckoutTagMethod checks out a given tag
	CheckoutTagMethod
	// CheckoutBranchMethod checks out a given branch
	CheckoutBranchMethod
	// CheckoutPRMergeBranchMethod checks out a MR/PR (when merge branch is available)
	CheckoutPRMergeBranchMethod
	// CheckoutPRDiffFileMethod  checks out a MR/PR (when a diff file is available)
	CheckoutPRDiffFileMethod
	// CheckoutPRManualMergeMethod check out a Merge Request using manual merge

	CheckoutPRManualMergeMethod
	// CheckoutForkPRManualMergeMethod checks out a PR using manual merge
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
	do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error
}

// X: required parameter
// !: used to identify checkout strategy
// _: optional parameter
// |==================================================================================|
// | params\strat| commit | tag | branch | manualMR | manualPR | autoMerge | autoDiff |
// | commit      |  X  !  |     |        |  X       |          |           |          |
// | tag         |        |  X !|        |          |          |           |          |
// | branch      |  _     |  _  |  X !   |  X       |  X       |  X        |          |
// | branchDest  |        |     |        |  X       |  X       |           |  X       |
// | PRRepoURL   |        |     |        |      !   |  X !     |    !      |    !     |
// | PRID        |        |     |        |          |          |           |    !     |
// | mergeBranch |        |     |        |          |          |  X !      |          |
// |==================================================================================|

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

	isFork := isFork(cfg.RepositoryURL, cfg.PRRepositoryURL)
	isPrivateFork := isPrivate(cfg.PRRepositoryURL) && isFork
	if !cfg.ManualMerge || isPrivateFork {
		if cfg.PRMergeBranch != "" {
			return CheckoutPRMergeBranchMethod
		}

		return CheckoutPRDiffFileMethod
	}

	if isFork {
		return CheckoutForkPRManualMergeMethod
	}

	return CheckoutPRManualMergeMethod
}

func createCheckoutStrategy(checkoutMethod CheckoutMethod, cfg Config, patch patchSource) (checkoutStrategy, error) {
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
			params, err := NewBranchParams(cfg.Branch)
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
			patchFile, err := patch.getDiffPath(cfg.BuildURL, cfg.BuildAPIToken)
			if err != nil {
				return nil, fmt.Errorf("merging PR (automatic) failed, there is no Pull Request branch and could not download diff file: %v", err)
			}

			return checkoutPRDiffFile{
				baseBranch: cfg.BranchDest,
				patchFile:  patchFile,
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

func selectFetchOptions(checkoutStrategy CheckoutMethod, cloneDepth int, isTag bool) fetchOptions {
	switch checkoutStrategy {
	case CheckoutCommitMethod, CheckoutBranchMethod:
		return fetchOptions{
			depth:   cloneDepth,
			allTags: isTag,
		}
	case CheckoutTagMethod:
		return fetchOptions{
			depth:   cloneDepth,
			allTags: true,
		}
	case CheckoutPRMergeBranchMethod, CheckoutPRDiffFileMethod:
		return fetchOptions{
			depth:   cloneDepth,
			allTags: false,
		}
	case CheckoutPRManualMergeMethod, CheckoutForkPRManualMergeMethod, CheckoutNoneMethod:
		return fetchOptions{}
	default:
		return fetchOptions{}
	}
}

func selectFallbacks(checkoutStrategy CheckoutMethod, fetchOpts fetchOptions) fallbackRetry {
	switch checkoutStrategy {
	case CheckoutCommitMethod, CheckoutTagMethod:
		{
			if !fetchOpts.IsFullDepth() {
				return simpleUnshallow{}
			}

			return nil
		}
	case CheckoutPRMergeBranchMethod:
		{
			if !fetchOpts.IsFullDepth() {
				return resetUnshallow{}
			}

			return nil
		}
	case CheckoutPRManualMergeMethod, CheckoutForkPRManualMergeMethod, CheckoutBranchMethod, CheckoutPRDiffFileMethod:
		return nil
	default:
		return nil
	}
}
