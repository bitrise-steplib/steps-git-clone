package gitclone

import (
	"errors"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
)

//
// checkoutNone
type checkoutNone struct{}

func (c checkoutNone) Validate() error {
	return nil
}

func (c checkoutNone) Do(gitCmd git.Git, fetchOptions fetchOptions) error {
	return nil
}

//
// checkoutCommit
type checkoutCommit struct {
	commit string
}

func (c checkoutCommit) Validate() error {
	if strings.TrimSpace(c.commit) == "" {
		return errors.New("no commit hash specified")
	}

	return nil
}

func (c checkoutCommit) Do(gitCmd git.Git, fetchOptions fetchOptions) error {
	// Fetch then checkout
	// No branch specified for fetch
	if err := fetch(gitCmd, fetchOptions, nil); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{arg: c.commit}, simpleUnshallowFunc); err != nil {
		return err
	}

	return nil
}

//
// checkoutBranch
type checkoutBranch struct {
	branch string
}

func (c checkoutBranch) Validate() error {
	if strings.TrimSpace(c.branch) == "" {
		return errors.New("no branch specified")
	}

	return nil
}

func (c checkoutBranch) Do(gitCmd git.Git, fetchOptions fetchOptions) error {
	branchRef := *newOriginFetchRef(branchRefPrefix + c.branch)
	if err := fetchInitialBranch(gitCmd, branchRef, fetchOptions); err != nil {
		return err
	}

	return nil
}

//
// checkoutTag
type checkoutTag struct {
	tag    string
	branch *string // Optional
}

func (c checkoutTag) Validate() error {
	if strings.TrimSpace(c.tag) == "" {
		return errors.New("no tag specified")
	}

	return nil
}

func (c checkoutTag) Do(gitCmd git.Git, fetchOptions fetchOptions) error {
	var branchRef *fetchRef
	if c.branch != nil {
		branchRef = newOriginFetchRef(branchRefPrefix + *c.branch)
	}

	if err := fetch(gitCmd, fetchOptions, branchRef); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{arg: c.tag}, simpleUnshallowFunc); err != nil {
		return err
	}

	return nil
}
