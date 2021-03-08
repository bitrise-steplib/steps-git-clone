package gitclone

import (
	"strings"
)

type CheckoutMethod int

const (
	InvalidCheckoutMethod CheckoutMethod = iota
	CheckoutNoneMeyhod
	CheckoutCommitMethod
	CheckoutTagMethod
	CheckoutBranchMethod
	CheckoutPRMergeBranchMethod
	CheckoutPRDiffFileMethod
	CheckoutPRManualMergeMethod
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

// CommitParams are parameters to check out a given commit (In addition to the repository URL)
type CommitParams struct {
	Commit string
}

// NewCommitParams validates and returns a new CommitParams
func NewCommitParams(commit string) (*CommitParams, error) {
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("commit checkout strategy can not be used, no commit hash specified")
	}

	return &CommitParams{
		Commit: commit,
	}, nil
}

// BranchParams are parameters to check out a given branch (In addition to the repository URL)
type BranchParams struct {
	Branch string
	Commit *string
}

// NewBranchParams validates and returns a new BranchParams
func NewBranchParams(branch string, commit *string) (*BranchParams, error) {
	if strings.TrimSpace(branch) == "" {
		return nil, NewParameterValidationError("branch checkout strategy can not be used, no branch specified")
	}
	if commit != nil && strings.TrimSpace(*commit) == "" {
		return nil, NewParameterValidationError("branch checkout strategy can not be used, no commit specified")
	}

	return &BranchParams{
		Branch: branch,
		Commit: commit,
	}, nil
}

// TagParams are parameters to checko out a given tag
type TagParams struct {
	Tag    string
	Branch *string
}

// NewTagParams validates and returns a new TagParams
func NewTagParams(tag string, branch *string) (*TagParams, error) {
	if strings.TrimSpace(tag) == "" {
		return nil, NewParameterValidationError("tag checkout strategy can not be used, no tag specified")
	}
	if branch != nil && strings.TrimSpace(*branch) == "" {
		return nil, NewParameterValidationError("tag checkout strategy can not be used, branch non nil but empty")
	}

	return &TagParams{
		Tag:    tag,
		Branch: branch,
	}, nil
}

// PRManualMergeParams are parameters to check out a Merge Request if no merge branch or diff file is avavilable
type PRManualMergeParams struct {
	// Source
	HeadBranch, Commit string
	// Target
	BaseBranch string
}

//NewPRManualMergeParams  validates and returns a new PRManualMergeParams
func NewPRManualMergeParams(headBranch, commit, baseBranch string) (*PRManualMergeParams, error) {
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used, no head branch specified")
	}
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used, no head branch commit hash specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("manual PR merge checkout strategy can not be used, no base branch specified")
	}

	return &PRManualMergeParams{
		HeadBranch: headBranch,
		Commit:     commit,
		BaseBranch: baseBranch,
	}, nil
}

// ForkPRManualMergeParams are parameters to check out a Pull Request if no merge branch or diff file is available
type ForkPRManualMergeParams struct {
	// Source
	HeadBranch, HeadRepoURL string
	// Target
	BaseBranch string
}

// NewForkPRManualMergeParams validates and returns a new ForkPRManualMergeParams
func NewForkPRManualMergeParams(headBranch, forkRepoURL, baseBranch string) (*ForkPRManualMergeParams, error) {
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge checkout strategy can not be used, no head branch specified")
	}
	if strings.TrimSpace(forkRepoURL) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge chekout strategy can not be used, no base repository URL specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("manual PR (fork) merge checkout strategy can not be used, no base branch specified")
	}

	return &ForkPRManualMergeParams{
		HeadBranch:  headBranch,
		HeadRepoURL: forkRepoURL,
		BaseBranch:  baseBranch,
	}, nil
}

// PRMergeBranchParams are parameters to check out a Merge/Pull Request if merge branch is available
type PRMergeBranchParams struct {
	BaseBranch string
	// Merge branch contains the changes premerged by the Git provider
	MergeBranch string
}

// NewPRMergeBranchParams validates and returns a new PRMergeBranchParams
func NewPRMergeBranchParams(baseBranch, mergeBranch string) (*PRMergeBranchParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR merge branch based checkout strategy can not be used, no base branch specified")
	}
	if strings.TrimSpace(mergeBranch) == "" {
		return nil, NewParameterValidationError("PR merge branch based checkout strategy can not be used, no merge branch specified")
	}

	return &PRMergeBranchParams{
		BaseBranch:  baseBranch,
		MergeBranch: mergeBranch,
	}, nil
}

// PRDiffFileParams are parameters to check out a Merge/Pull Request if a diff file is available
type PRDiffFileParams struct {
	BaseBranch string
	PRID       uint
}

// NewPRDiffFileParams validates and returns a new PRDiffFile
func NewPRDiffFileParams(baseBranch string, PRID uint) (*PRDiffFileParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewParameterValidationError("PR diff file based checkout strategy can not be used, base branch specified")
	}

	return &PRDiffFileParams{
		BaseBranch: baseBranch,
		PRID:       PRID,
	}, nil
}
