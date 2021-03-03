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

const bracnhPrefix = "refs/heads"

type fetchRef struct {
	Remote, Ref string
}

func newFetchRef(ref string) *fetchRef {
	return &fetchRef{
		Remote: defaultRemoteName,
		Ref:    ref,
	}
}

type fetchRetry struct {
	didUnshallow bool
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
	if ref != nil && strings.HasPrefix(ref.Ref, bracnhPrefix) {
		branch = strings.TrimPrefix(ref.Ref, bracnhPrefix)
	}

	if err := runner.RunWithRetry(gitCmd.Fetch(opts...)); err != nil {
		return handleCheckoutError(
			listBranches(gitCmd),
			fetchFailedTag,
			fmt.Errorf("fetch failed, error: %v", err),
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
				fmt.Errorf("fetch (unshallow) failed, error: %v", err),
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

func checkoutOnly(gitCmd git.Git, arg checkoutArg, fetchRetry fetchRetry) *step.Error {
	if err := runner.Run(gitCmd.Checkout(arg.Arg)); err != nil {
		if fetchRetry.didUnshallow {
			return newStepError(
				"checkout_unshallow_failed",
				fmt.Errorf("checkout failed (%s), error: %v", arg.Arg, err),
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
			fmt.Errorf("checkout failed (%s), error: %v", arg.Arg, err),
			"Checkout has failed",
			branch,
		)
	}

	return nil
}

func merge(gitCmd git.Git, branch string) *step.Error {
	if err := runner.Run(gitCmd.Merge("origin/" + branch)); err != nil {
		return newStepError(
			"update_branch_failed",
			fmt.Errorf("updating branch (merge) failed %q: %v", branch, err),
			"Updating branch failed",
		)
	}

	return nil
}
