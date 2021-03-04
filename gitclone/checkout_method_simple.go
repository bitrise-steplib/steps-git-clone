package gitclone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

type checkoutStrategy interface {
	Validate() error
	Do(gitCmd git.Git) *step.Error
}

//
// checkoutNone
type checkoutNone struct {
	ShouldUpdateSubmodules bool
}

func (c checkoutNone) Validate() error {
	return nil
}

func (c checkoutNone) Do(gitCmd git.Git) *step.Error {
	if c.ShouldUpdateSubmodules {
		return updateSubmodules(gitCmd)
	}

	return nil
}

//
// checkoutCommit
type checkoutCommit struct {
	Commit, branch         string
	FetchTraits            fetchTraits
	ShouldUpdateSubmodules bool
}

func (c checkoutCommit) Validate() error {
	if strings.TrimSpace(c.Commit) == "" {
		return errors.New("no commit hash specified")
	}

	return nil
}

func (c checkoutCommit) Do(gitCmd git.Git) *step.Error {
	// Fetch then checkout
	// No branch specified for fetch
	if err := fetch(gitCmd, c.FetchTraits, nil, nil); err != nil {
		return newStepError(
			"a", err, "aaaa",
		)
	}

	if err := checkoutOnly(gitCmd, checkoutArg{Arg: c.Commit}, func(cerr error) error {
		log.Warnf("Checkout failed, error: %v\nUnshallow...", cerr)
		if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
			return newStepError(
				"fetch_unshallow_failed",
				fmt.Errorf("fetch (unshallow) failed: %v", err),
				"Fetching with unshallow parameter has failed",
			)
		}

		return nil
	}); err != nil {
		return newStepError("a", err, "aaa")
	}

	if c.ShouldUpdateSubmodules {
		return updateSubmodules(gitCmd)
	}

	return nil
}

//
// checkoutBranch
type checkoutBranch struct {
	Branch                 string
	FetchTraits            fetchTraits
	ShouldUpdateSubmodules bool
}

func (c checkoutBranch) Validate() error {
	if strings.TrimSpace(c.Branch) == "" {
		return errors.New("no branch specified")
	}

	return nil
}

func (c checkoutBranch) Do(gitCmd git.Git) *step.Error {
	branchRef := *newOriginFetchRef(branchRefPrefix + c.Branch)
	if err := fetchInitialBranch(gitCmd, branchRef, c.FetchTraits); err != nil {
		return err
	}

	if c.ShouldUpdateSubmodules {
		return updateSubmodules(gitCmd)
	}

	return nil
}

//
// checkoutTag
type checkoutTag struct {
	Tag                    string
	Branch                 *string // Optional
	FetchTraits            fetchTraits
	ShouldUpdateSubmodules bool
}

func (c checkoutTag) Validate() error {
	if strings.TrimSpace(c.Tag) == "" {
		return errors.New("no tag specified")
	}

	return nil
}

func (c checkoutTag) Do(gitCmd git.Git) *step.Error {
	var branchRef *fetchRef
	if c.Branch != nil {
		branchRef = newOriginFetchRef(branchRefPrefix + *c.Branch)
	}

	if err := fetch(gitCmd, c.FetchTraits, branchRef, nil); err != nil {
		return err
	}

	if err := checkoutOnly(gitCmd, checkoutArg{Arg: c.Tag}, func(cerr error) error {
		log.Warnf("Checkout failed, error: %v\nUnshallow...", cerr)
		if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
			return newStepError(
				"fetch_unshallow_failed",
				fmt.Errorf("fetch (unshallow) failed: %v", err),
				"Fetching with unshallow parameter has failed",
			)
		}

		return nil
	}); err != nil {
		return newStepError("a", err, "aaa")
	}

	if c.ShouldUpdateSubmodules {
		return updateSubmodules(gitCmd)
	}

	return nil
}
