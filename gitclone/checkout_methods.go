package gitclone

import (
	"errors"
	"strings"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command/git"
)

func checkoutStateStrangler(gitCmd git.Git, cfg Config) *step.Error {
	var checkoutMethod checkoutMethod
	isPR := cfg.PRRepositoryURL != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if !isPR && cfg.Commit == "" && cfg.Branch == "" && cfg.Tag == "" {
		checkoutMethod = checkoutNone{
			ShouldUpdateSubmodules: cfg.UpdateSubmodules,
		}
	}
	if !isPR && cfg.Commit != "" {
		checkoutMethod = checkoutCommit{
			Commit: cfg.Commit,
			Branch: cfg.Branch,
			FetchTraits: fetchTraits{
				Depth: cfg.CloneDepth,
				Tags:  cfg.Tag != "",
			},
			ShouldUpdateSubmodules: cfg.UpdateSubmodules,
		}
	}

	if checkoutMethod != nil {
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

// checkoutCommit
type checkoutCommit struct {
	Commit, Branch         string
	FetchTraits            fetchTraits
	ShouldUpdateSubmodules bool
}

func (c checkoutCommit) Validate() error {
	if strings.TrimSpace(c.Commit) == "" {
		return errors.New("precondition failed, no commit hash specified")
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
