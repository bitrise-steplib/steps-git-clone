//go:generate stringer -type=CheckoutMethod

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

	// CheckoutCommitMethod checks out a given commit on a given branch
	CheckoutCommitMethod

	// CheckoutTagMethod checks out a given tag
	CheckoutTagMethod

	// CheckoutBranchMethod checks out a given branch's head when a commit hash is not available
	CheckoutBranchMethod

	// CheckoutPRMergeBranchMethod creates the merge result by fetching the merge ref from the destination repo (if available)
	CheckoutPRMergeBranchMethod

	// CheckoutPRDiffFileMethod  creates the merge result of a PR/MR by applying the diff manually (if available)
	CheckoutPRDiffFileMethod

	// CheckoutPRManualMergeMethod creates the merge result by merging the PR/MR branch into the destination branch
	CheckoutPRManualMergeMethod

	// CheckoutHeadBranchCommitMethod checks out the PR/MR branch head without merging it into the destination branch
	CheckoutHeadBranchCommitMethod

	// CheckoutForkCommitMethod checks out the PR from the fork repo (if accessible)
	CheckoutForkCommitMethod
)

const privateForkAuthWarning = `May fail due to missing authentication as Pull Request opened from a private fork.
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

	// getBuildTriggerRef returns ref to the commit/branch/tag that triggered the build.
	// For simple checkout strategies the returned ref will be HEAD (after running 'do').
	// However, a PR checkout strategy may create a (temporary) merge commit, so the merged state can be tested.
	// In this case the returned ref will point to the Source branch (or a commit on the Source branch).
	getBuildTriggerRef() string
}

// X: required parameter
// !: used to identify checkout strategy
// _: optional parameter
// |=========================================================================|
// | params\strat| commit | tag | branch | manualMR | headBranch | diffFile  |
// | commit      |  X  !  |     |        |  _/X     |  _/X       |           |
// | tag         |        |  X !|        |          |            |           |
// | branch      |  _     |  _  |  X !   |  X       |            |           |
// | branchDest  |        |     |        |  X  !    |  X !       |  X  !     |
// | PRRepoURL   |        |     |        |  _       |            |           |
// | PRID        |        |     |        |          |            |           |
// | mergeBranch |        |     |        |          |    !       |           |
// | headBranch  |        |     |        |          |  X         |           |
// |=========================================================================|

func selectCheckoutMethod(cfg Config, patch patchSource) (CheckoutMethod, string) {
	isPR := cfg.PRSourceRepositoryURL != "" || cfg.PRDestBranch != "" || cfg.PRMergeBranch != ""
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

	isFork := isFork(cfg.RepositoryURL, cfg.PRSourceRepositoryURL)
	isPrivateSourceRepo := isPrivate(cfg.PRSourceRepositoryURL)
	isPublicFork := isFork && !isPrivateSourceRepo

	// PR: check out the head of the PR branch
	if !cfg.ShouldMergePR {
		if cfg.PRHeadBranch != "" {
			// Git server provides a head ref (e.g. refs/pull/2/head), so even if this is a true Pull Request
			// from a fork (which we might not be able to access), we can check out the PR head through the destination repo
			return CheckoutHeadBranchCommitMethod, ""
		}

		if !isFork {
			// It's a Merge Request, we have access to the MR branch
			return CheckoutCommitMethod, ""
		}

		if isPublicFork {
			// Even though it's not an MR, we can access the source branch
			return CheckoutForkCommitMethod, ""
		}

		// Fallback (Bitbucket only): it's a PR from a fork we can't access, so we fetch the PR patch file through
		// the API and apply the diff manually
		if cfg.BuildURL != "" {
			patchFile := getPatchFile(patch, cfg.BuildURL, cfg.BuildAPIToken)
			if patchFile != "" {
				log.Infof("Merging Pull Request despite the option to disable merging, as it is opened from a private fork.")
				return CheckoutPRDiffFileMethod, patchFile
			}
		}

		log.Warnf(privateForkAuthWarning)
		return CheckoutForkCommitMethod, ""
	}

	// PR: check out the merge result (merging the PR branch into the destination branch)
	if cfg.PRMergeBranch != "" {
		// Merge ref (such as refs/pull/2/merge) is available in the destination repo, we can access that
		// even if the PR source is a private repo
		return CheckoutPRMergeBranchMethod, ""
	}

	// Fallback (Bitbucket only): fetch the PR patch file through the API and apply the diff manually
	if cfg.BuildURL != "" {
		patchFile := getPatchFile(patch, cfg.BuildURL, cfg.BuildAPIToken)
		if patchFile != "" {
			return CheckoutPRDiffFileMethod, patchFile
		}
	}

	// As a last resort, fetch target + PR branches and do a manual merge
	// This is not ideal because the merge requires fetched branch histories. If the fetch is too shallow,
	// the merge is going to fail with "refusing to merge unrelated histories"
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
			branchRef := ""
			if cfg.Branch != "" {
				branchRef = refsHeadsPrefix + cfg.Branch
			}

			params, err := NewCommitParams(cfg.Commit, branchRef, "")
			if err != nil {
				return nil, err
			}

			return checkoutCommit{
				params: *params,
			}, nil
		}
	case CheckoutTagMethod:
		{
			params, err := NewTagParams(cfg.Tag)
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
			params, err := NewPRMergeRefParams(cfg.PRMergeBranch, cfg.PRHeadBranch)
			if err != nil {
				return nil, err
			}

			return checkoutPRMergeRef{
				params: *params,
			}, nil
		}
	case CheckoutPRDiffFileMethod:
		{
			prManualMergeStrategy, err := createCheckoutStrategy(CheckoutPRManualMergeMethod, cfg, patchFile)
			if err != nil {
				return nil, err
			}

			params, err := NewPRDiffFileParams(cfg.PRDestBranch, prManualMergeStrategy)
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
			if isFork(cfg.RepositoryURL, cfg.PRSourceRepositoryURL) {
				prRepositoryURL = cfg.PRSourceRepositoryURL
			}

			params, err := NewPRManualMergeParams(cfg.Branch, cfg.Commit, prRepositoryURL, cfg.PRDestBranch)
			if err != nil {
				return nil, err
			}

			return checkoutPRManualMerge{
				params: *params,
			}, nil
		}
	case CheckoutHeadBranchCommitMethod:
		{
			headBranchRef := refsPrefix + cfg.PRHeadBranch // ref/pull/2/head
			params, err := NewCommitParams(cfg.Commit, headBranchRef, "")
			if err != nil {
				return nil, err
			}

			return checkoutCommit{
				params: *params,
			}, nil
		}
	case CheckoutForkCommitMethod:
		{
			sourceBranchRef := refsHeadsPrefix + cfg.Branch
			params, err := NewCommitParams(cfg.Commit, sourceBranchRef, cfg.PRSourceRepositoryURL)
			if err != nil {
				return nil, err
			}

			return checkoutCommit{
				params: *params,
			}, nil
		}
	default:
		return nil, fmt.Errorf("invalid checkout strategy selected")
	}
}

func selectFetchOptions(method CheckoutMethod, cloneDepth int, fetchTags, fetchSubmodules bool, filterTree bool) fetchOptions {
	// If cloneDepth is 0, that means the user did not set a value for it,
	// so we will determine the correct value based on the checkout method.
	if cloneDepth == 0 {
		cloneDepth = idealDefaultCloneDepth(method)
	}

	opts := fetchOptions{
		limitDepth:      cloneDepth > 0,
		depth:           cloneDepth,
		tags:            fetchTags,
		fetchSubmodules: fetchSubmodules,
	}
	opts = selectFilterTreeFetchOption(method, opts, filterTree)

	return opts
}

func selectFilterTreeFetchOption(method CheckoutMethod, opts fetchOptions, filterTree bool) fetchOptions {
	if !filterTree {
		return opts
	}

	switch method {
	case CheckoutCommitMethod,
		CheckoutTagMethod,
		CheckoutBranchMethod,
		CheckoutHeadBranchCommitMethod,
		CheckoutForkCommitMethod:
		{
			opts.filterTree = true
			return opts
		}
	case CheckoutNoneMethod,
		CheckoutPRMergeBranchMethod,
		CheckoutPRManualMergeMethod,
		CheckoutPRDiffFileMethod:
		{
			return opts
		}
	default:
		panic(fmt.Sprintf("implementation missing for enum value %T", method))
	}
}

func idealDefaultCloneDepth(method CheckoutMethod) int {
	const defaultCloneDepth = 50
	const shallowCloneDepth = 1

	if method == CheckoutPRManualMergeMethod {
		return defaultCloneDepth
	} else {
		return shallowCloneDepth
	}
}

func selectFallbacks(method CheckoutMethod, fetchOpts fetchOptions) fallbackRetry {
	if fetchOpts.IsFullDepth() {
		return nil
	}

	unshallowFetchOpts := unshallowFetchOptions{
		tags:            fetchOpts.tags,
		fetchSubmodules: fetchOpts.fetchSubmodules,
	}

	switch method {
	case CheckoutNoneMethod,
		CheckoutBranchMethod,        // the given branch's tip will be checked out, no need to unshallow
		CheckoutPRMergeBranchMethod: // there is no manual merge in this case, so the shallow checkout can't be a problem
		{
			return nil
		}
	case CheckoutCommitMethod,
		CheckoutTagMethod,
		CheckoutHeadBranchCommitMethod,
		CheckoutForkCommitMethod:
		{
			return simpleUnshallow{
				traits: unshallowFetchOpts,
			}
		}
	case CheckoutPRManualMergeMethod,
		CheckoutPRDiffFileMethod:
		{
			return resetUnshallow{
				traits: unshallowFetchOpts,
			}
		}
	default:
		panic(fmt.Sprintf("implementation missing for enum value %T", method))
	}
}

func isPRCheckout(method CheckoutMethod) bool {
	switch method {
	case CheckoutNoneMethod,
		CheckoutCommitMethod,
		CheckoutTagMethod,
		CheckoutBranchMethod:
		{
			return false
		}
	case CheckoutPRMergeBranchMethod,
		CheckoutPRDiffFileMethod,
		CheckoutPRManualMergeMethod,
		CheckoutHeadBranchCommitMethod,
		CheckoutForkCommitMethod:
		{
			return true
		}
	default:
		panic(fmt.Sprintf("implementation missing for enum value %T", method))
	}
}
