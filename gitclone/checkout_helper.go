package gitclone

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

type fetchParams struct {
	branch  string
	remote  string
	options fetchOptions
}

type mergeParams struct {
	arg      string
	fallback fallbackRetry
}

type fetchOptions struct {
	// Sets '--tags' flag
	// From https://git-scm.com/docs/fetch-options/2.29.0#Documentation/fetch-options.txt---allTags:
	// "Fetch all tags from the remote (i.e., fetch remote tags refs/tags/* into local tags with the same name),
	// in addition to whatever else would otherwise be fetched"
	allTags bool
	// Sets '--depth' flag
	// More info: https://git-scm.com/docs/fetch-options/2.29.0#Documentation/fetch-options.txt---depthltdepthgt
	depth int
}

func (t fetchOptions) IsFullDepth() bool {
	return t.depth == 0
}

const (
	refsHeadsPrefix = "refs/heads/"
	refsPrefix      = "refs/"
)

func branchRefToTrackingBranch(branchRef string) string {
	trackingBranch := branchRef
	if strings.HasPrefix(branchRef, refsHeadsPrefix) {
		trackingBranch = strings.TrimPrefix(trackingBranch, refsHeadsPrefix)
	} else {
		trackingBranch = strings.TrimPrefix(trackingBranch, refsPrefix)
	}

	return trackingBranch
}

func fetch(gitCmd git.Git, remote string, ref string, trackingBranch string, traits fetchOptions) error {
	var opts []string
	if traits.depth != 0 {
		opts = append(opts, "--depth="+strconv.Itoa(traits.depth))
	}
	if traits.allTags {
		opts = append(opts, "--tags")
	}
	if ref != "" {
		fetchArg := ref
		if trackingBranch != "" {
			fetchArg = fmt.Sprintf("%s:%s", ref, trackingBranch)
		}
		opts = append(opts, remote, fetchArg)
	}

	// Not neccessarily a branch, can be tag too
	branch := ""
	if strings.HasPrefix(ref, refsHeadsPrefix) {
		branch = strings.TrimPrefix(ref, refsHeadsPrefix)
	}

	if err := runner.RunWithRetry(func() *command.Model {
		return gitCmd.Fetch(opts...)
	}); err != nil {
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

func checkoutWithCustomRetry(gitCmd git.Git, arg string, retry fallbackRetry) error {
	if cErr := runner.Run(gitCmd.Checkout(arg)); cErr != nil {
		if retry != nil {
			log.Warnf("Checkout failed (%s): %v", arg, cErr)
			if err := retry.do(gitCmd); err != nil {
				return err
			}

			return runner.Run(gitCmd.Checkout(arg))
		}

		return fmt.Errorf("checkout failed (%s): %v", arg, cErr)
	}

	return nil
}

func fetchInitialBranch(gitCmd git.Git, remote string, branchRef string, fetchTraits fetchOptions) error {
	trackingBranch := branchRefToTrackingBranch(branchRef)
	if err := fetch(gitCmd, remote, branchRef, trackingBranch, fetchTraits); err != nil {
		return err
	}

	if err := checkoutWithCustomRetry(gitCmd, trackingBranch, nil); err != nil {
		return handleCheckoutError(
			listBranches(gitCmd),
			checkoutFailedTag,
			err,
			"Checkout has failed",
			trackingBranch,
		)
	}

	return nil
}

func mergeWithCustomRetry(gitCmd git.Git, arg string, retry fallbackRetry) error {
	if mErr := runner.Run(gitCmd.Merge(arg)); mErr != nil {
		if retry != nil {
			log.Warnf("Merge failed (%s): %v", arg, mErr)
			if err := retry.do(gitCmd); err != nil {
				return err
			}

			return runner.Run(gitCmd.Merge(arg))
		}

		return fmt.Errorf("merge failed (%s): %v", arg, mErr)
	}

	return nil
}

func fetchAndMerge(gitCmd git.Git, fetchParam fetchParams, mergeParam mergeParams) error {
	headBranchRef := refsHeadsPrefix + fetchParam.branch
	if err := fetch(gitCmd, fetchParam.remote, headBranchRef, "", fetchParam.options); err != nil {
		return nil
	}

	return mergeWithCustomRetry(gitCmd, mergeParam.arg, mergeParam.fallback)
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
