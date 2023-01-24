package gitclone

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

// checkoutNone
type checkoutNone struct{}

func (c checkoutNone) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	return nil
}

func (c checkoutNone) getBuildTriggerRef() string {
	return ""
}

// CommitParams are parameters to check out a given commit (In addition to the repository URL)
type CommitParams struct {
	Commit        string
	BranchRef     string
	SourceRepoURL string // optional
}

// NewCommitParams validates and returns a new CommitParams
func NewCommitParams(commit, branchRef, sourceRepoURL string) (*CommitParams, error) {
	if strings.TrimSpace(commit) == "" {
		return nil, NewParameterValidationError("commit checkout strategy can not be used: no commit hash specified")
	}

	return &CommitParams{
		Commit:        commit,
		BranchRef:     branchRef,
		SourceRepoURL: sourceRepoURL,
	}, nil
}

// checkoutCommit
type checkoutCommit struct {
	params CommitParams
}

func (c checkoutCommit) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	remote := originRemoteName
	if c.params.SourceRepoURL != "" {
		remote = forkRemoteName
		if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.SourceRepoURL)); err != nil {
			return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.SourceRepoURL, err)
		}
	}

	if err := fetch(gitCmd, remote, c.params.BranchRef, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.Commit, fallback); err != nil {
		return err
	}

	return nil
}

func (c checkoutCommit) getBuildTriggerRef() string {
	return c.params.Commit
}

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
	if err := fetchInitialBranch(gitCmd, originRemoteName, c.localRef(), fetchOptions); err != nil {
		return err
	}

	return nil
}

func (c checkoutBranch) getBuildTriggerRef() string {
	return c.localRef()
}

func (c checkoutBranch) localRef() string {
	return refsHeadsPrefix + c.params.Branch
}

// TagParams are parameters to check out a given tag
type TagParams struct {
	Tag string
}

// NewTagParams validates and returns a new TagParams
func NewTagParams(tag string) (*TagParams, error) {
	if strings.TrimSpace(tag) == "" {
		return nil, NewParameterValidationError("tag checkout strategy can not be used: no tag specified")
	}

	return &TagParams{
		Tag: tag,
	}, nil
}

// checkoutTag
type checkoutTag struct {
	params TagParams
}

func (c checkoutTag) do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error {
	ref := fmt.Sprintf("%s:%s", c.ref(), c.ref())
	if err := fetch(gitCmd, originRemoteName, ref, fetchOptions); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, c.params.Tag, fallback); err != nil {
		return err
	}

	return nil
}

func (c checkoutTag) getBuildTriggerRef() string {
	return c.ref()
}

func (c checkoutTag) ref() string {
	return fmt.Sprintf("refs/tags/%s", c.params.Tag)
}
