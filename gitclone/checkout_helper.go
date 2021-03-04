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

var simpleUnshallowFunc func(git.Git, error) error = func(gitCmd git.Git, perr error) error {
	log.Warnf("Checkout failed, error: %v\nUnshallow...", perr)

	return runner.RunWithRetry(gitCmd.Fetch("--unshallow"))
}

func fetch(gitCmd git.Git, traits fetchTraits, ref *fetchRef) *step.Error {
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

	return nil
}

type checkoutArg struct {
	Arg      string
	IsBranch bool
}

func checkoutWithCustomRetry(gitCmd git.Git, arg checkoutArg, retryFunc func(error) error) error {
	if cerr := runner.Run(gitCmd.Checkout(arg.Arg)); cerr != nil {
		if retryFunc != nil {
			if err := retryFunc(cerr); err != nil {
				return err
			}

			return runner.Run(gitCmd.Checkout(arg.Arg))
		}

		return cerr
	}

	return nil
}

func fetchInitialBranch(gitCmd git.Git, ref fetchRef, fetchTraits fetchTraits) *step.Error {
	branch := strings.TrimPrefix(ref.Ref, branchRefPrefix)
	// Fetch then checkout
	if err := fetch(gitCmd, fetchTraits, &ref); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{Arg: branch, IsBranch: true}, nil); err != nil {
		return handleCheckoutError(
			listBranches(gitCmd),
			checkoutFailedTag,
			fmt.Errorf("checkout failed (%s): %v", branch, err),
			"Checkout has failed",
			branch,
		)
	}

	// Update branch: 'git fetch' followed by a 'git merge' is the same as 'git pull'.
	remoteBranch := fmt.Sprintf("%s/%s", defaultRemoteName, branch)
	if err := runner.Run(gitCmd.Merge(remoteBranch)); err != nil {
		return newStepError(
			"update_branch_failed",
			fmt.Errorf("updating branch (merge) failed %q: %v", branch, err),
			"Updating branch failed",
		)
	}

	return nil
}

func mergeWithCustomRetry(gitCmd git.Git, arg string, retryFunc func(gitCmd git.Git, merr error) error) error {
	if merr := runner.Run(gitCmd.Merge(arg)); merr != nil {
		if retryFunc != nil {
			if err := retryFunc(gitCmd, merr); err != nil {
				return err
			}

			return runner.Run(gitCmd.Merge(arg))
		}

		return fmt.Errorf("merging %q: %v", arg, merr)
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
