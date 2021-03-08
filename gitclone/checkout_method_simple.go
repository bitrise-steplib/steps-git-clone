package gitclone

import (
	"errors"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

//
// checkoutNone
type checkoutNone struct{}

func (c checkoutNone) do(gitCmd git.Git, fetchOptions fetchOptions, fallbacks fallbacks) error {
	return nil
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

//
// checkoutCommit
type checkoutCommit struct {
	params CommitParams
}

func (c checkoutCommit) Validate() error {
	if strings.TrimSpace(c.params.Commit) == "" {
		return errors.New("no commit hash specified")
	}

	return nil
}

func (c checkoutCommit) do(gitCmd git.Git, fetchOptions fetchOptions, fallbacks fallbacks) error {
	// Fetch then checkout
	// No branch specified for fetch
	if err := fetch(gitCmd, fetchOptions, nil); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{arg: c.params.Commit}, fallbacks.checkout); err != nil {
		return err
	}

	return nil
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

//
// checkoutBranch
type checkoutBranch struct {
	params BranchParams
}

func (c checkoutBranch) do(gitCmd git.Git, fetchOptions fetchOptions, fallbacks fallbacks) error {
	branchRef := *newOriginFetchRef(branchRefPrefix + c.params.Branch)
	if err := fetchInitialBranch(gitCmd, branchRef, fetchOptions); err != nil {
		return err
	}

	return nil
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

//
// checkoutTag
type checkoutTag struct {
	params TagParams
}

func (c checkoutTag) do(gitCmd git.Git, fetchOptions fetchOptions, fallbacks fallbacks) error {
	var branchRef *fetchRef
	if c.params.Branch != nil {
		branchRef = newOriginFetchRef(branchRefPrefix + *c.params.Branch)
	}

	if err := fetch(gitCmd, fetchOptions, branchRef); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{arg: c.params.Tag}, fallbacks.checkout); err != nil {
		return err
	}

	return nil
}
