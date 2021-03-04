package gitclone

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bitrise-io/bitrise-init/step"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

type fetchTraits struct {
	// Fetch tags ("--tags")
	Tags bool
	// Clone depth ("--depth=")
	Depth int
}

func (t fetchTraits) IsFullDepth() bool {
	return t.Depth == 0
}

const branchRefPrefix = "refs/heads/"

type fetchRef struct {
	Remote, Ref string
}

func newOriginFetchRef(ref string) *fetchRef {
	return &fetchRef{
		Remote: defaultRemoteName,
		Ref:    ref,
	}
}

type fetchRetry struct {
	didUnshallow bool
}

func fetch(gitCmd git.Git, traits fetchTraits, ref *fetchRef, callback func(fetchRetry) *step.Error) *step.Error {
	var opts []string
	if traits.Depth != 0 {
		opts = append(opts, "--depth="+strconv.Itoa(traits.Depth))
	}
	if traits.Tags {
		opts = append(opts, "--tags")
	}
	if ref != nil {
		opts = append(opts, ref.Remote, ref.Ref)
	}

	// Not neccessarily a branch, can be tag too
	branch := ""
	if ref != nil && strings.HasPrefix(ref.Ref, branchRefPrefix) {
		branch = strings.TrimPrefix(ref.Ref, branchRefPrefix)
	}

	if err := runner.RunWithRetry(gitCmd.Fetch(opts...)); err != nil {
		return handleCheckoutError(
			listBranches(gitCmd),
			fetchFailedTag,
			fmt.Errorf("fetch failed: %v", err),
			"Fetching repository has failed",
			branch,
		)
	}

	if callback != nil {
		err := callback(fetchRetry{didUnshallow: false})
		if err == nil {
			return nil
		}
		if traits.IsFullDepth() {
			return err
		}

		log.Warnf("Checkout failed, error: %v\nUnshallow...", err)
		if err := runner.RunWithRetry(gitCmd.Fetch("--unshallow")); err != nil {
			return newStepError(
				"fetch_unshallow_failed",
				fmt.Errorf("fetch (unshallow) failed: %v", err),
				"Fetching with unshallow parameter has failed",
			)
		}

		return callback(fetchRetry{didUnshallow: true})
	}

	return nil
}

type checkoutArg struct {
	Arg      string
	IsBranch bool
}

func checkoutOnly(gitCmd git.Git, arg checkoutArg, fetchRetry *fetchRetry) *step.Error {
	if err := runner.Run(gitCmd.Checkout(arg.Arg)); err != nil {
		if fetchRetry != nil && fetchRetry.didUnshallow {
			return newStepError(
				"checkout_unshallow_failed",
				fmt.Errorf("checkout failed (%s): %v", arg.Arg, err),
				"Checkout after unshallow fetch has failed",
			)
		}

		branch := ""
		if arg.IsBranch {
			branch = arg.Arg
		}
		return handleCheckoutError(
			listBranches(gitCmd),
			checkoutFailedTag,
			fmt.Errorf("checkout failed (%s): %v", arg.Arg, err),
			"Checkout has failed",
			branch,
		)
	}

	return nil
}

func fetchInitialBranch(gitCmd git.Git, ref fetchRef, fetchTraits fetchTraits) *step.Error {
	branch := strings.TrimPrefix(ref.Ref, branchRefPrefix)
	// Fetch then checkout
	if err := fetch(gitCmd, fetchTraits, &ref, nil); err != nil {
		return err
	}

	if err := checkoutOnly(gitCmd, checkoutArg{Arg: branch, IsBranch: true}, nil); err != nil {
		return err
	}

	// Update branch: 'git fetch' followed by a 'git merge' is the same as 'git pull'.
	if err := runner.Run(gitCmd.Merge("origin/" + branch)); err != nil {
		return newStepError(
			"update_branch_failed",
			fmt.Errorf("updating branch (merge) failed %q: %v", branch, err),
			"Updating branch failed",
		)
	}

	return nil
}

func mergeCommit(gitCmd git.Git, commit string) *step.Error {
	if err := runner.Run(gitCmd.Merge(commit)); err != nil {
		return newStepError(
			"merge_failed",
			fmt.Errorf("merge failed %q: %v", commit, err),
			"Merge branch failed",
		)
	}

	return nil
}

func updateSubmodules(gitCmd git.Git) *step.Error {
	if err := runner.Run(gitCmd.SubmoduleUpdate()); err != nil {
		return newStepError(
			updateSubmodelFailedTag,
			fmt.Errorf("submodule update: %v", err),
			"Updating submodules has failed",
		)
	}

	return nil
}

func detachHead(gitCmd git.Git) *step.Error {
	if err := runner.Run(gitCmd.Checkout("--detach")); err != nil {
		return newStepError(
			"detach_head_failed",
			fmt.Errorf("detaching head failed: %v", err),
			"Detaching head failed",
		)
	}

	return nil
}
