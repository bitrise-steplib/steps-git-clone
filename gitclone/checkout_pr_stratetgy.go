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
// checkoutMergeRequestManual
type checkoutMergeRequestManual struct {
	// Source
	branchHead, commit string
	// Destination
	branchBase string
	// Other
	fetchTraits            fetchTraits
	shouldUpdateSubmodules bool
}

func (c checkoutMergeRequestManual) Validate() error {
	if strings.TrimSpace(c.branchHead) == "" {
		return errors.New("no head branch specified")
	}
	if strings.TrimSpace(c.commit) == "" {
		return errors.New("no head branch commit hash specified")
	}
	if strings.TrimSpace(c.branchBase) == "" {
		return errors.New("no base branch specified")
	}

	return nil
}

func (c checkoutMergeRequestManual) Do(gitCmd git.Git) *step.Error {
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
// checkoutForkPullRequestManual
type checkoutForkPullRequestManual struct {
	// Source
	branchFork, forkRepoURL string
	// Destination
	branchBase string
	// Other
	fetchTraits            fetchTraits
	shouldUpdateSubmodules bool
}

func (c checkoutForkPullRequestManual) Validate() error {
	if strings.TrimSpace(c.branchFork) == "" {
		return errors.New("no head branch specified")
	}
	if strings.TrimSpace(c.branchBase) == "" {
		return errors.New("no base repository URL specified")
	}
	if strings.TrimSpace(c.branchBase) == "" {
		return errors.New("no base branch specified")
	}

	return nil
}

func (c checkoutForkPullRequestManual) Do(gitCmd git.Git) *step.Error {
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

//
// checkoutPullRequestAutoMergeBranch
type checkoutPullRequestAutoMergeBranch struct {
	baseBranch string
	// Merge branch contains the changes already merged
	mergeBranch string
	// Other
	fetchTraits            fetchTraits
	shouldUpdateSubmodules bool
}

func (c checkoutPullRequestAutoMergeBranch) Validate() error {
	if strings.TrimSpace(c.baseBranch) == "" {
		return errors.New("no base branch specified")
	}
	if strings.TrimSpace(c.mergeBranch) == "" {
		return errors.New("no merge branch specified")
	}

	return nil
}

func mergeMergeBranch(gitCmd git.Git, branchName string, resetFunc func(merr error) error) error {
	if merr := runner.Run(gitCmd.Merge(branchName)); merr != nil {
		if resetFunc != nil {
			//log.Warnf("Merge failed, error: %v\nReset repository, then unshallow...", err)
			if err := resetFunc(merr); err != nil {
				return err
			}

			return runner.Run(gitCmd.Merge(branchName))
		}

		return fmt.Errorf("merging %q: %v", branchName, merr)
	}

	return nil
}

func (c checkoutPullRequestAutoMergeBranch) Do(gitCmd git.Git) *step.Error {
	// Check out initial branch (fetchInitialBranch part1)
	// `git "fetch" "origin" "refs/heads/master"`
	baseBranchRef := newOriginFetchRef(branchRefPrefix + c.baseBranch)
	if err := fetch(gitCmd, c.fetchTraits, baseBranchRef, nil); err != nil {
		return err
	}

	// `git "fetch" "origin" "refs/pull/7/head:pull/7"`
	// Does not apply clone depth (legacy)
	headBranchRef := newOriginFetchRef(fetchArg(c.mergeBranch))
	if err := fetch(gitCmd, fetchTraits{}, headBranchRef, nil); err != nil {
		return err
	}

	// Check out initial branch (fetchInitialBranch part2)
	// `git "checkout" "master"`
	// `git "merge" "origin/master"`
	if err := checkoutOnly(gitCmd, checkoutArg{Arg: c.baseBranch, IsBranch: true}, nil); err != nil {
		return err
	}
	remoteBaseBranch := fmt.Sprintf("%s/%s", defaultRemoteName, c.baseBranch)
	if err := runner.Run(gitCmd.Merge(remoteBaseBranch)); err != nil {
		return newStepError(
			"a",
			err,
			"aaaa",
		)
	}

	// `git "merge" "pull/7"`
	var resetFunc func(error) error
	if !c.fetchTraits.IsFullDepth() {
		resetFunc = func(merr error) error {
			log.Warnf("Merge failed: %v\nReset repository, then unshallow...", merr)

			if err := resetRepo(gitCmd); err != nil {
				return fmt.Errorf("reset repository: %v", err)
			}
			if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
				return fmt.Errorf("fetch failed: %v", err)
			}

			return nil
		}
	}
	if err := mergeMergeBranch(gitCmd, mergeArg(c.mergeBranch), resetFunc); err != nil {
		return newStepError(
			"a",
			fmt.Errorf("merge failed: %s", err),
			"aaa",
		)
	}

	if c.shouldUpdateSubmodules {
		if err := updateSubmodules(gitCmd); err != nil {
			return err
		}
	}

	return detachHead(gitCmd)
}

//
// checkoutPullRequestAutoDiffFile
type checkoutPullRequestAutoDiffFile struct {
	baseBranch, patch string
	// Other
	fetchTraits            fetchTraits
	shouldUpdateSubmodules bool
}

func (c checkoutPullRequestAutoDiffFile) Validate() error {
	if strings.TrimSpace(c.baseBranch) == "" {
		return errors.New("no base branch specified")
	}

	return nil
}

func (c checkoutPullRequestAutoDiffFile) Do(gitCmd git.Git) *step.Error {
	baseBranchRef := newOriginFetchRef(branchRefPrefix + c.baseBranch)
	if err := fetch(gitCmd, c.fetchTraits, baseBranchRef, nil); err != nil {
		return err
	}

	if err := runner.Run(gitCmd.Checkout(c.baseBranch)); err != nil {
		return newStepError(
			"a",
			fmt.Errorf("checkout failed (%s): %v", c.baseBranch, err),
			"aaa",
		)
	}

	if err := runner.Run(gitCmd.Apply(c.patch)); err != nil {
		return newStepError(
			"a",
			fmt.Errorf("can't apply patch (%s): %v", c.patch, err),
			"aaa",
		)
	}

	if c.shouldUpdateSubmodules {
		if err := updateSubmodules(gitCmd); err != nil {
			return err
		}
	}

	return detachHead(gitCmd)
}
