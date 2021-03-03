package gitclone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command/git"
)

func choose(cfg Config) checkoutMethod {
	defaultFetchTraits := fetchTraits{
		Depth: cfg.CloneDepth,
		Tags:  cfg.Tag != "",
	}

	isPR := cfg.PRRepositoryURL != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if !isPR {
		if cfg.Commit == "" && cfg.Branch == "" && cfg.Tag == "" {
			return checkoutNone{
				ShouldUpdateSubmodules: cfg.UpdateSubmodules,
			}
		}
		if cfg.Commit != "" {
			return checkoutCommit{
				Commit: cfg.Commit,
				// Branch: cfg.Branch,
				FetchTraits:            defaultFetchTraits,
				ShouldUpdateSubmodules: cfg.UpdateSubmodules,
			}
		}
		if cfg.Tag != "" {
			return checkoutTag{
				Tag:    cfg.Tag,
				Branch: cfg.Branch,
				FetchTraits: fetchTraits{
					Depth: cfg.CloneDepth,
					Tags:  true,
				},
				ShouldUpdateSubmodules: cfg.UpdateSubmodules,
			}
		}
		if cfg.Tag == "" && cfg.Branch != "" {
			return checkoutBranch{
				Branch:                 cfg.Branch,
				FetchTraits:            defaultFetchTraits,
				ShouldUpdateSubmodules: cfg.UpdateSubmodules,
			}
		}
	}

	return nil
}

func checkoutStateStrangler(gitCmd git.Git, cfg Config) *step.Error {
	checkoutMethod := choose(cfg)
	if checkoutMethod != nil {
		if err := checkoutMethod.Validate(); err != nil {
			return newStepError(
				"checkout_method_select",
				fmt.Errorf("Checkout method can not be used (%T): %v", checkoutMethod, err),
				"Internal error",
			)
		}
		if err := checkoutMethod.Do(gitCmd); err != nil {
			return err
		}
	} else {
		if err := checkoutState(gitCmd, cfg); err != nil {
			return err
		}
	}

	return nil
}

type checkoutMethod interface {
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
	Commit, Branch         string
	FetchTraits            fetchTraits
	ShouldUpdateSubmodules bool
}

func (c checkoutCommit) Validate() error {
	if strings.TrimSpace(c.Commit) == "" {
		return errors.New("precondition not satisfied, no commit hash specified")
	}

	return nil
}

func (c checkoutCommit) Do(gitCmd git.Git) *step.Error {
	// Fetch then checkout
	// No branch specified for fetch
	if err := fetch(gitCmd, c.FetchTraits, nil, func(fetchRetry fetchRetry) *step.Error {
		return checkoutOnly(gitCmd, checkoutArg{Arg: c.Commit}, fetchRetry)
	}); err != nil {
		return err
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
		return errors.New("precondition not satisfied, no branch specified")
	}

	return nil
}

func (c checkoutBranch) Do(gitCmd git.Git) *step.Error {
	branchRef := fmt.Sprintf("%s/%s", bracnhPrefix, c.Branch)
	if err := fetch(gitCmd, c.FetchTraits, newFetchRef(branchRef), func(fetchRetry fetchRetry) *step.Error {
		return checkoutOnly(gitCmd, checkoutArg{Arg: c.Branch, IsBranch: true}, fetchRetry)
	}); err != nil {
		return err
	}

	// Update branch: 'git fetch' followed by a 'git merge' is the same as 'git pull'.
	if err := merge(gitCmd, c.Branch); err != nil {
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
	Tag, Branch            string
	FetchTraits            fetchTraits
	ShouldUpdateSubmodules bool
}

func (c checkoutTag) Validate() error {
	if strings.TrimSpace(c.Tag) == "" {
		return errors.New("precondition not satisifed, no tag specified")
	}

	return nil
}

func (c checkoutTag) Do(gitCmd git.Git) *step.Error {
	branchRef := fmt.Sprintf("%s/%s", bracnhPrefix, c.Branch)
	if err := fetch(gitCmd, c.FetchTraits, newFetchRef(branchRef), func(fetchRetry fetchRetry) *step.Error {
		return checkoutOnly(gitCmd, checkoutArg{Arg: c.Tag}, fetchRetry)
	}); err != nil {
		return err
	}

	if c.ShouldUpdateSubmodules {
		return updateSubmodules(gitCmd)
	}

	return nil
}
