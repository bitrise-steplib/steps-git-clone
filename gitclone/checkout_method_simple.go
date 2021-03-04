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

func (c checkoutNone) Do(gitCmd git.Git) error {
	return nil
}

//
// checkoutCommit
type checkoutCommit struct {
	Commit      string
	FetchTraits fetchTraits
}

func (c checkoutCommit) Validate() error {
	if strings.TrimSpace(c.Commit) == "" {
		return errors.New("no commit hash specified")
	}

	return nil
}

func (c checkoutCommit) Do(gitCmd git.Git) error {
	// Fetch then checkout
	// No branch specified for fetch
	if err := fetch(gitCmd, c.FetchTraits, nil); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{Arg: c.Commit}, simpleUnshallowFunc); err != nil {
		return err
	}

	return nil
}

//
// checkoutBranch
type checkoutBranch struct {
	Branch      string
	FetchTraits fetchTraits
}

func (c checkoutBranch) Validate() error {
	if strings.TrimSpace(c.Branch) == "" {
		return errors.New("no branch specified")
	}

	return nil
}

func (c checkoutBranch) Do(gitCmd git.Git) error {
	branchRef := *newOriginFetchRef(branchRefPrefix + c.Branch)
	if err := fetchInitialBranch(gitCmd, branchRef, c.FetchTraits); err != nil {
		return err
	}

	return nil
}

//
// checkoutTag
type checkoutTag struct {
	Tag         string
	Branch      *string // Optional
	FetchTraits fetchTraits
}

func (c checkoutTag) Validate() error {
	if strings.TrimSpace(c.Tag) == "" {
		return errors.New("no tag specified")
	}

	return nil
}

func (c checkoutTag) Do(gitCmd git.Git) error {
	var branchRef *fetchRef
	if c.Branch != nil {
		branchRef = newOriginFetchRef(branchRefPrefix + *c.Branch)
	}

	if err := fetch(gitCmd, c.FetchTraits, branchRef); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{Arg: c.Tag}, simpleUnshallowFunc); err != nil {
		return err
	}

	return nil
}
