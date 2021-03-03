package gitclone

import (
	"errors"
	"strings"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

//
// checkoutPRManualMerge
type checkoutPRManualMerge struct {
	// Source
	Branch, Commit string
	// Destination
	BranchDest             string
	fetchTraits            fetchTraits
	ShouldUpdateSubmodules bool
}

func (c checkoutPRManualMerge) Validate() error {
	if strings.TrimSpace(c.Branch) == "" {
		return errors.New("no source bracnh specified")
	}
	if strings.TrimSpace(c.Commit) == "" {
		return errors.New("no source commit hash specified")
	}
	if strings.TrimSpace(c.BranchDest) == "" {
		return errors.New("no destiantion branch specified")
	}

	return nil
}

func (c checkoutPRManualMerge) Do(gitCmd git.Git) *step.Error {
	destBranchRef := newOriginFetchRef(branchRefPrefix + c.BranchDest)
	if err := fetch(gitCmd, c.fetchTraits, destBranchRef, nil); err != nil {
		return err
	}
	if err := checkoutOnly(gitCmd, checkoutArg{Arg: c.BranchDest}, fetchRetry{}); err != nil {
		return err
	}
	if err := mergeBranch(gitCmd, c.BranchDest); err != nil {
		return nil
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	sourceBranchRef := newOriginFetchRef(branchRefPrefix + c.Branch)
	if err := fetch(gitCmd, c.fetchTraits, sourceBranchRef, func(fetchRetry) *step.Error {
		return mergeCommit(gitCmd, c.Commit)
	}); err != nil {
		return nil
	}

	if c.ShouldUpdateSubmodules {
		if err := updateSubmodules(gitCmd); err != nil {
			return err
		}
	}

	return detachHead(gitCmd)
}
