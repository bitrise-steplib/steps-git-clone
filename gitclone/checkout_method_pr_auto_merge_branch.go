package gitclone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

//
// checkoutPullRequestAutoMergeBranch
type checkoutPullRequestAutoMergeBranch struct {
	baseBranch string
	// Merge branch contains the changes already merged
	mergeBranch string
	// Other
	fetchTraits fetchTraits
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

func (c checkoutPullRequestAutoMergeBranch) Do(gitCmd git.Git) error {
	// Check out initial branch (fetchInitialBranch part1)
	// `git "fetch" "origin" "refs/heads/master"`
	baseBranchRef := newOriginFetchRef(branchRefPrefix + c.baseBranch)
	if err := fetch(gitCmd, c.fetchTraits, baseBranchRef); err != nil {
		return err
	}

	// `git "fetch" "origin" "refs/pull/7/head:pull/7"`
	// Does not apply clone depth (legacy)
	headBranchRef := newOriginFetchRef(fetchArg(c.mergeBranch))
	if err := fetch(gitCmd, fetchTraits{}, headBranchRef); err != nil {
		return err
	}

	// Check out initial branch (fetchInitialBranch part2)
	// `git "checkout" "master"`
	// `git "merge" "origin/master"`
	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{Arg: c.baseBranch, IsBranch: true}, nil); err != nil {
		return err
	}
	remoteBaseBranch := fmt.Sprintf("%s/%s", defaultRemoteName, c.baseBranch)
	if err := runner.Run(gitCmd.Merge(remoteBaseBranch)); err != nil {
		return err
	}

	// `git "merge" "pull/7"`
	var resetFunc func(git.Git, error) error
	if !c.fetchTraits.IsFullDepth() {
		resetFunc = func(gitCmd git.Git, merr error) error {
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
	if err := mergeWithCustomRetry(gitCmd, mergeArg(c.mergeBranch), resetFunc); err != nil {
		return err
	}

	return detachHead(gitCmd)
}
