package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
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
	// CheckoutHeadBranchMethod checks out a MR/PR head branch only, without merging into base branch
	CheckoutHeadBranchMethod
	// CheckoutForkBranchMethod checks out a PR source branch, without merging
	CheckoutForkBranchMethod
)

const privateForkAuthWarning = `May fail due to missing authentication as Pull/Merge Request opened from a private fork.
A git hosting provider head branch or a diff file is unavailable.`

// ParameterValidationError is returned when there is missing or malformatted parameter for a given parameter set
type ParameterValidationError struct {
	ErrorString string
}

// Error ...
func (e ParameterValidationError) Error() string {
	return e.ErrorString
}

// NewParameterValidationError returns a new ValidationError
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
// |==========================================================================================================================|
// | params\strat| commit | tag | branch | manualMR | manualPR | autoMerge | autoDiff | noMergeHeadBranch | noMergeForkBranch |
// | commit      |  X  !  |     |        |  X       |          |           |          |                   |                   |
// | tag         |        |  X !|        |          |          |           |          |                   |                   |
// | branch      |  _     |  _  |  X !   |  X       |  X       |  X        |          |                   |  X                |
// | branchDest  |        |     |        |  X       |  X       |           |  X       |                   |                   |
// | PRRepoURL   |        |     |        |      !   |  X !     |    !      |    !     |                   |  X !              |
// | PRID        |        |     |        |          |          |           |    !     |                   |    !              |
// | mergeBranch |        |     |        |          |          |  X !      |          |    !              |                   |
// | headBranch  |        |     |        |          |          |           |          |  X !              |                   |
// |==========================================================================================================================|

func selectCheckoutMethod(cfg Config, patch patchSource) (CheckoutMethod, string) {
	isPR := cfg.PRRepositoryURL != "" || cfg.BranchDest != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if !isPR {
		if cfg.Commit != "" {
			return CheckoutCommitMethod, ""
		}

		if cfg.Tag != "" {
			return CheckoutTagMethod, ""
		}

		if cfg.Branch != "" {
			return CheckoutBranchMethod, ""
		}

		return CheckoutNoneMethod, ""
	}

	isFork := isFork(cfg.RepositoryURL, cfg.PRRepositoryURL)
	isPrivateSourceRepo := isPrivate(cfg.PRRepositoryURL)
	isPrivateFork := isFork && isPrivateSourceRepo
	isPublicFork := isFork && !isPrivateSourceRepo

	if !cfg.ShouldMergePR {
		if cfg.PRHeadBranch != "" {
			return CheckoutHeadBranchMethod, ""
		}

		if !isFork {
			return CheckoutBranchMethod, ""
		}

		if isPublicFork {
			return CheckoutForkBranchMethod, ""
		}

		if cfg.BuildURL != "" {
			patchFile := getPatchFile(patch, cfg.BuildURL, cfg.BuildAPIToken)
			if patchFile != "" {
				log.Infof("Merging Pull/Merge Request despite the option to disable merging, as it is opened from a private fork.")

				return CheckoutPRDiffFileMethod, patchFile
			}
		}

		log.Warnf(privateForkAuthWarning)
		return CheckoutForkBranchMethod, ""
	}

	if !cfg.ManualMerge || isPrivateFork {
		if cfg.PRMergeBranch != "" {
			return CheckoutPRMergeBranchMethod, ""
		}

		if cfg.BuildURL != "" {
			patchFile := getPatchFile(patch, cfg.BuildURL, cfg.BuildAPIToken)
			if patchFile != "" {
				return CheckoutPRDiffFileMethod, patchFile
			}
		}

		log.Warnf(privateForkAuthWarning)
		return CheckoutPRManualMergeMethod, ""
	}

	return CheckoutPRManualMergeMethod, ""
}

func getPatchFile(patch patchSource, buildURL, buildAPIToken string) string {
	if patch != nil {
		patchFile, err := patch.getDiffPath(buildURL, buildAPIToken)
		if err != nil {
			log.Warnf("Diff file unavailable: %v", err)
		} else {
			return patchFile
		}
	}

	return ""
}

func createCheckoutStrategy(checkoutMethod CheckoutMethod, cfg Config, patchFile string) (checkoutStrategy, error) {
	switch checkoutMethod {
	case CheckoutNoneMethod:
		{
			return checkoutNone{}, nil
		}
	case CheckoutCommitMethod:
		{
			params, err := NewCommitParams(cfg.Commit, cfg.Branch)
			if err != nil {
				return nil, err
			}

			return checkoutCommit{
				params: *params,
			}, nil
		}
	case CheckoutTagMethod:
		{
			params, err := NewTagParams(cfg.Tag, cfg.Branch)
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
			prManualMergeStrategy, err := createCheckoutStrategy(CheckoutPRManualMergeMethod, cfg, patchFile)
			if err != nil {
				return nil, err
			}

			params, err := NewPRDiffFileParams(cfg.BranchDest, prManualMergeStrategy)
			if err != nil {
				return nil, err
			}

			return checkoutPRDiffFile{
				params:    *params,
				patchFile: patchFile,
			}, nil
		}
	case CheckoutPRManualMergeMethod:
		{
			prRepositoryURL := ""
			if isFork(cfg.RepositoryURL, cfg.PRRepositoryURL) {
				prRepositoryURL = cfg.PRRepositoryURL
			}

			params, err := NewPRManualMergeParams(cfg.Branch, cfg.Commit, prRepositoryURL, cfg.BranchDest)
			if err != nil {
				return nil, err
			}

			return checkoutPRManualMerge{
				params: *params,
			}, nil
		}
	case CheckoutHeadBranchMethod:
		{
			params, err := NewCheckoutHeadBranchParams(cfg.PRHeadBranch)
			if err != nil {
				return nil, err
			}

			return checkoutHeadBranch{
				params: *params,
			}, nil
		}
	case CheckoutForkBranchMethod:
		{
			params, err := NewCheckoutForkBranchParams(cfg.Branch, cfg.PRRepositoryURL)
			if err != nil {
				return nil, err
			}

			return checkoutForkBranch{
				params: *params,
			}, nil
		}
	default:
		return nil, fmt.Errorf("invalid checkout strategy selected")
	}

}

func selectFetchOptions(checkoutStrategy CheckoutMethod, cloneDepth int, fetchAllTags bool) fetchOptions {
	opts := fetchOptions{
		depth:   cloneDepth,
		allTags: false,
	}

	switch checkoutStrategy {
	case CheckoutCommitMethod, CheckoutBranchMethod:
		opts.allTags = fetchAllTags
	case CheckoutTagMethod:
		opts.allTags = true
	default:
	}

	return opts
}

func selectFallbacks(checkoutStrategy CheckoutMethod, fetchOpts fetchOptions) fallbackRetry {
	if fetchOpts.IsFullDepth() {
		return nil
	}

	switch checkoutStrategy {
	case CheckoutBranchMethod, CheckoutHeadBranchMethod, CheckoutForkBranchMethod:
		// the given branch's tip will be checked out, no need to unshallow
		return nil
	case CheckoutCommitMethod, CheckoutTagMethod:
		return simpleUnshallow{}
	case CheckoutPRMergeBranchMethod, CheckoutPRManualMergeMethod, CheckoutPRDiffFileMethod:
		return resetUnshallow{}
	default:
		return nil
	}
}
