package gitcloneparams

import (
	"strings"
)

// ValidationError is returned when there is missing or malformatted parameter for a given parameter set
type ValidationError struct {
	ErrorString string
}

// Error ...
func (e ValidationError) Error() string {
	return e.ErrorString
}

// NewValidationError ...
func NewValidationError(msg string) error {
	return ValidationError{ErrorString: msg}
}

// CommitParams ...
type CommitParams struct {
	Commit string
}

// NewCommitParams ...
func NewCommitParams(commit string) (*CommitParams, error) {
	if strings.TrimSpace(commit) == "" {
		return nil, NewValidationError("commit checkout strategy can not be used, no commit hash specified")
	}

	return &CommitParams{
		Commit: commit,
	}, nil
}

// BranchParams
type BranchParams struct {
	Branch string
	Commit *string
}

// NewBranchParams
func NewBranchParams(branch string, commit *string) (*BranchParams, error) {
	if strings.TrimSpace(branch) == "" {
		return nil, NewValidationError("branch checkout strategy can not be used, no branch specified")
	}
	if commit != nil && strings.TrimSpace(*commit) == "" {
		return nil, NewValidationError("branch checkout strategy can not be used, no commit specified")
	}

	return &BranchParams{
		Branch: branch,
		Commit: commit,
	}, nil
}

// TagParams
type TagParams struct {
	Tag    string
	Branch *string
}

// NewTagParams
func NewTagParams(tag string, branch *string) (*TagParams, error) {
	if strings.TrimSpace(tag) == "" {
		return nil, NewValidationError("tag checkout strategy can not be used, no tag specified")
	}
	if branch != nil && strings.TrimSpace(*branch) == "" {
		return nil, NewValidationError("tag checkout strategy can not be used, branch non nil but empty")
	}

	return &TagParams{
		Tag:    tag,
		Branch: branch,
	}, nil
}

// PRManualMergeParams
type PRManualMergeParams struct {
	// Source
	HeadBranch, Commit string
	// Target
	BaseBranch string
}

//NewPRManualMergeParams
func NewPRManualMergeParams(headBranch, commit, baseBranch string) (*PRManualMergeParams, error) {
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewValidationError("manual PR merge checkout strategy can not be used, no head branch specified")
	}
	if strings.TrimSpace(commit) == "" {
		return nil, NewValidationError("manual PR merge checkout strategy can not be used, no head branch commit hash specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewValidationError("manual PR merge checkout strategy can not be used, no base branch specified")
	}

	return &PRManualMergeParams{
		HeadBranch: headBranch,
		Commit:     commit,
		BaseBranch: baseBranch,
	}, nil
}

// ForkPRManualMergeParams
type ForkPRManualMergeParams struct {
	// Source
	HeadBranch, HeadRepoURL string
	// Target
	BaseBranch string
}

// NewForkPRManualMergeParams
func NewForkPRManualMergeParams(headBranch, forkRepoURL, baseBranch string) (*ForkPRManualMergeParams, error) {
	if strings.TrimSpace(headBranch) == "" {
		return nil, NewValidationError("manual PR (fork) merge checkout strategy can not be used, no head branch specified")
	}
	if strings.TrimSpace(forkRepoURL) == "" {
		return nil, NewValidationError("manual PR (fork) merge chekout strategy can not be used, no base repository URL specified")
	}
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewValidationError("manual PR (fork) merge checkout strategy can not be used, no base branch specified")
	}

	return &ForkPRManualMergeParams{
		HeadBranch:  headBranch,
		HeadRepoURL: forkRepoURL,
		BaseBranch:  baseBranch,
	}, nil
}

// PRMergeBranchParams
type PRMergeBranchParams struct {
	BaseBranch string
	// Merge branch contains the changes premerged by the Git provider
	MergeBranch string
}

// NewPRMergeBranchParams
func NewPRMergeBranchParams(baseBranch, mergeBranch string) (*PRMergeBranchParams, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewValidationError("PR merge branch based checkout strategy can not be used, no base branch specified")
	}
	if strings.TrimSpace(mergeBranch) == "" {
		return nil, NewValidationError("PR merge branch based checkout strategy can not be used, no merge branch specified")
	}

	return &PRMergeBranchParams{
		BaseBranch:  baseBranch,
		MergeBranch: mergeBranch,
	}, nil
}

// PRDiffFileParams
type PRDiffFile struct {
	BaseBranch string
	PRID       uint
}

// NewPRDiffFileParams
func NewPRDiffFileParams(baseBranch string, PRID uint) (*PRDiffFile, error) {
	if strings.TrimSpace(baseBranch) == "" {
		return nil, NewValidationError("PR diff file based checkout strategy can not be used, base branch specified")
	}

	return &PRDiffFile{
		BaseBranch: baseBranch,
		PRID:       PRID,
	}, nil
}
