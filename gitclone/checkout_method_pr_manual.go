package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

//
// checkoutPRManualMerge
type checkoutPRManualMerge struct {
	params PRManualMergeParams
}

func (c checkoutPRManualMerge) do(gitCmd git.Git, fetchOptions fetchOptions) error {
	// Fetch and checkout base (target) branch
	baseBranchRef := *newOriginFetchRef(branchRefPrefix + c.params.BaseBranch)
	if err := fetchInitialBranch(gitCmd, baseBranchRef, fetchOptions); err != nil {
		return err
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	// Fetch and merge
	headBranchRef := newOriginFetchRef(branchRefPrefix + c.params.HeadBranch)
	if err := fetch(gitCmd, fetchOptions, headBranchRef); err != nil {
		return nil
	}

	var unshallowFunc func(git.Git, error) error
	if !fetchOptions.IsFullDepth() {
		unshallowFunc = simpleUnshallowFunc
	}

	if err := mergeWithCustomRetry(gitCmd, c.params.Commit, unshallowFunc); err != nil {
		return err
	}

	return detachHead(gitCmd)
}

//
// checkoutForkPRManualMerge
type checkoutForkPRManualMerge struct {
	params ForkPRManualMergeParams
}

func (c checkoutForkPRManualMerge) do(gitCmd git.Git, fetchOptions fetchOptions) error {
	// Fetch and checkout base branch
	baseBranchRef := *newOriginFetchRef(branchRefPrefix + c.params.BaseBranch)
	if err := fetchInitialBranch(gitCmd, baseBranchRef, fetchOptions); err != nil {
		return err
	}

	commitHash, err := runner.RunForOutput(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	const forkRemoteName = "fork"
	// Add fork remote
	if err := runner.Run(gitCmd.RemoteAdd(forkRemoteName, c.params.HeadRepoURL)); err != nil {
		return fmt.Errorf("adding remote fork repository failed (%s): %v", c.params.HeadRepoURL, err)
	}

	// Fetch + merge fork branch
	forkBranchRef := fetchRef{
		remote: forkRemoteName,
		ref:    branchRefPrefix + c.params.HeadBranch,
	}
	remoteForkBranch := fmt.Sprintf("%s/%s", forkRemoteName, c.params.HeadBranch)
	if err := fetch(gitCmd, fetchOptions, &forkBranchRef); err != nil {
		return err
	}

	if err := mergeWithCustomRetry(gitCmd, remoteForkBranch, simpleUnshallowFunc); err != nil {
		return err
	}

	return detachHead(gitCmd)
}
