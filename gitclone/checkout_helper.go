package gitclone

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

type fetchOptions struct {
	// Fetch tags ("--tags")
	tags bool
	// Clone depth ("--depth=")
	depth int
}

func (t fetchOptions) IsFullDepth() bool {
	return t.depth == 0
}

const branchRefPrefix = "refs/heads/"

type fetchRef struct {
	remote, ref string
}

func newOriginFetchRef(ref string) *fetchRef {
	return &fetchRef{
		remote: defaultRemoteName,
		ref:    ref,
	}
}

var simpleUnshallowFunc func(git.Git, error) error = func(gitCmd git.Git, perr error) error {
	log.Warnf("Checkout failed, error: %v\nUnshallow...", perr)

	return runner.RunWithRetry(gitCmd.Fetch("--unshallow"))
}

func fetch(gitCmd git.Git, traits fetchOptions, ref *fetchRef) error {
	var opts []string
	if traits.depth != 0 {
		opts = append(opts, "--depth="+strconv.Itoa(traits.depth))
	}
	if traits.tags {
		opts = append(opts, "--tags")
	}
	if ref != nil {
		opts = append(opts, ref.remote, ref.ref)
	}

	// Not neccessarily a branch, can be tag too
	branch := ""
	if ref != nil && strings.HasPrefix(ref.ref, branchRefPrefix) {
		branch = strings.TrimPrefix(ref.ref, branchRefPrefix)
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
	arg      string
	isBranch bool
}

func checkoutWithCustomRetry(gitCmd git.Git, arg checkoutArg, retryFunc func(git.Git, error) error) error {
	if cerr := runner.Run(gitCmd.Checkout(arg.arg)); cerr != nil {
		if retryFunc != nil {
			if err := retryFunc(gitCmd, cerr); err != nil {
				return err
			}

			return runner.Run(gitCmd.Checkout(arg.arg))
		}

		return cerr
	}

	return nil
}

func fetchInitialBranch(gitCmd git.Git, ref fetchRef, fetchTraits fetchOptions) error {
	branch := strings.TrimPrefix(ref.ref, branchRefPrefix)
	// Fetch then checkout
	if err := fetch(gitCmd, fetchTraits, &ref); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, checkoutArg{arg: branch, isBranch: true}, nil); err != nil {
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

		return fmt.Errorf("merge failed (%q): %v", arg, merr)
	}

	return nil
}

func detachHead(gitCmd git.Git) error {
	if err := runner.Run(gitCmd.Checkout("--detach")); err != nil {
		return newStepError(
			"detach_head_failed",
			fmt.Errorf("detaching head failed: %v", err),
			"Detaching head failed",
		)
	}

	return nil
}
