package gitclone

import (
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

//
// checkoutNone
type checkoutNone struct{}

func (c checkoutNone) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	return nil
}

//
// CommitParams are parameters to check out a given commit (In addition to the repository URL)
type CommitParams struct {
	Commit string
}

// NewCommitParams validates and returns a new CommitParams
func NewCommitParams(commit string) (*CommitParams, error) {
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("commit checkout strategy can not be used: no commit hash specified")
	}

	return &CommitParams{
		Commit: commit,
	}, nil
}

// checkoutCommit
type checkoutCommit struct {
	params CommitParams
}

func (c checkoutCommit) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	// Fetch then checkout
	// No branch specified for fetch
	if err := fetch(gitCmd, defaultRemoteName, nil, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.Commit, fallback); err != nil {
		return err
	}

	return nil
}

//
// BranchParams are parameters to check out a given branch (In addition to the repository URL)
type BranchParams struct {
	Branch string
}

// NewBranchParams validates and returns a new BranchParams
func NewBranchParams(branch string) (*BranchParams, error) {
	if strings.TrimSpace(branch) == "" {
		return nil, NewParameterValidationError("branch checkout strategy can not be used: no branch specified")
	}

	return &BranchParams{
		Branch: branch,
	}, nil
}

// checkoutBranch
type checkoutBranch struct {
	params BranchParams
}

func (c checkoutBranch) do(gitCmd git.Git, fetchOptions fetchOptions, _ fallbackRetry) error {
	branchRef := branchRefPrefix + c.params.Branch
	if err := fetchInitialBranch(gitCmd, defaultRemoteName, branchRef, fetchOptions); err != nil {
		return err
	}

	return nil
}

//
// TagParams are parameters to check out a given tag
type TagParams struct {
	Tag    string
	Branch *string
}

// NewTagParams validates and returns a new TagParams
func NewTagParams(tag string, branch *string) (*TagParams, error) {
	if strings.TrimSpace(tag) == "" {
		return nil, NewParameterValidationError("tag checkout strategy can not be used: no tag specified")
	}
	if branch != nil && strings.TrimSpace(*branch) == "" {
		return nil, NewParameterValidationError("tag checkout strategy can not be used: branch non nil but empty")
	}

	return &TagParams{
		Tag:    tag,
		Branch: branch,
	}, nil
}

// checkoutTag
type checkoutTag struct {
	params TagParams
}

func (c checkoutTag) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	var branchRefParam *string
	if c.params.Branch != nil {
		branchRef := branchRefPrefix + *c.params.Branch
		branchRefParam = &branchRef
	}

	if err := fetch(gitCmd, defaultRemoteName, branchRefParam, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.Tag, fallback); err != nil {
		return err
	}

	return nil
}
