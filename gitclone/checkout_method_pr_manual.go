package gitclone

import (
	"errors"
	"fmt"
	"strings"

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
	fetchTraits fetchTraits
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

func (c checkoutMergeRequestManual) Do(gitCmd git.Git) error {
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
	if err := fetch(gitCmd, c.fetchTraits, headBranchRef); err != nil {
		return nil
	}

	var unshallowFunc func(git.Git, error) error
	if !c.fetchTraits.IsFullDepth() {
		unshallowFunc = simpleUnshallowFunc
	}

	if err := mergeWithCustomRetry(gitCmd, c.commit, unshallowFunc); err != nil {
		return err
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
	fetchTraits fetchTraits
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

func (c checkoutForkPullRequestManual) Do(gitCmd git.Git) error {
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
		return fmt.Errorf("adding remote fork repository failed (%s): %v", c.forkRepoURL, err)
	}

	// Fetch + merge fork branch
	forkBranchRef := fetchRef{
		Remote: forkRemoteName,
		Ref:    branchRefPrefix + c.branchFork,
	}
	remoteForkBranch := fmt.Sprintf("%s/%s", forkRemoteName, c.branchFork)
	if err := fetch(gitCmd, c.fetchTraits, &forkBranchRef); err != nil {
		return err
	}

	if err := mergeWithCustomRetry(gitCmd, remoteForkBranch, simpleUnshallowFunc); err != nil {
		return err
	}

	return detachHead(gitCmd)
}
