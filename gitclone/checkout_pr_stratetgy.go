package gitclone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

//
// checkoutMRManualMerge
type checkoutMRManualMerge struct {
	// Source
	branchHead, commit string
	// Destination
	branchBase string
	// Other
	fetchTraits            fetchTraits
	shouldUpdateSubmodules bool
}

func (c checkoutMRManualMerge) Validate() error {
	if strings.TrimSpace(c.branchHead) == "" {
		return errors.New("no source branch specified")
	}
	if strings.TrimSpace(c.commit) == "" {
		return errors.New("no source commit hash specified")
	}
	if strings.TrimSpace(c.branchBase) == "" {
		return errors.New("no destiantion branch specified")
	}

	return nil
}

func (c checkoutMRManualMerge) Do(gitCmd git.Git) *step.Error {
	// Fetch and checkout base (target) branch
	baseBranchRef := *newOriginFetchRef(branchRefPrefix + c.branchBase)
	if err := fetchInitialBranch(gitCmd, baseBranchRef, c.fetchTraits); err != nil {
		return err
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	// Fetch and merge
	headBranchRef := newOriginFetchRef(branchRefPrefix + c.branchHead)
	if err := fetch(gitCmd, c.fetchTraits, headBranchRef, func(fetchRetry) *step.Error {
		if err := runner.Run(gitCmd.Merge(c.commit)); err != nil {
			return newStepError(
				"merge_failed",
				fmt.Errorf("merge failed %q: %v", c.commit, err),
				"Merge branch failed",
			)
		}

		return nil
	}); err != nil {
		return nil
	}

	if c.shouldUpdateSubmodules {
		if err := updateSubmodules(gitCmd); err != nil {
			return err
		}
	}

	return detachHead(gitCmd)
}

//
// checkoutForkPRManualMerge
type checkoutForkPRManualMerge struct {
	// Source
	branchFork, forkRepoURL string
	// Destination
	branchBase string
	// Other
	fetchTraits            fetchTraits
	shouldUpdateSubmodules bool
}

func (c checkoutForkPRManualMerge) Validate() error {
	if strings.TrimSpace(c.branchFork) == "" {
		return errors.New("no source branch specified")
	}
	if strings.TrimSpace(c.branchBase) == "" {
		return errors.New("no source repository URL specified")
	}
	if strings.TrimSpace(c.branchBase) == "" {
		return errors.New("no destiantion branch specified")
	}

	return nil
}

func (c checkoutForkPRManualMerge) Do(gitCmd git.Git) *step.Error {
	// Fetch and checkout base branch
	baseBranchRef := *newOriginFetchRef(branchRefPrefix + c.branchBase)
	if err := fetchInitialBranch(gitCmd, baseBranchRef, c.fetchTraits); err != nil {
		return err
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	const forkRemoteName = "fork"
	// Add fork remote
	if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.forkRepoURL)); err != nil {
		return newStepError(
			"add_remote_failed",
			fmt.Errorf("adding remote fork repository failed (%s): %v", c.forkRepoURL, err),
			"Adding remote fork repository failed",
		)
	}

	// Fetch + merge fork branch
	forkBranchRef := fetchRef{
		Remote: forkRemoteName,
		Ref:    branchRefPrefix + c.branchFork,
	}
	remoteForkBranch := fmt.Sprintf("%s/%s", forkRemoteName, c.branchFork)

	if err := fetch(gitCmd, c.fetchTraits, &forkBranchRef, func(fetchRetry) *step.Error {
		if err := runner.Run(gitCmd.Merge(remoteForkBranch)); err != nil {
			return newStepError(
				"merge_fork_failed",
				fmt.Errorf("merge failed (%s): %v", remoteForkBranch, err),
				"Merging fork remote branch failed",
			)
		}

		return nil
	}); err != nil {
		return err
	}

	if c.shouldUpdateSubmodules {
		if err := updateSubmodules(gitCmd); err != nil {
			return err
		}
	}

	return detachHead(gitCmd)
}
