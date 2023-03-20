package gitclone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

type fetchOptions struct {
	// Sets '--tags' or `--no-tags` flag
	// More info:
	// - https://git-scm.com/docs/git-fetch#Documentation/git-fetch.txt---tags
	// - https://git-scm.com/docs/git-fetch#Documentation/git-fetch.txt---no-tags
	tags bool
	// Sets '--depth' flag to the value of `depth` if `limitDepth` is true, doesn't set the flag otherwise
	// More info: https://git-scm.com/docs/fetch-options/2.29.0#Documentation/fetch-options.txt---depthltdepthgt
	limitDepth bool
	depth      int
	// Sets '--no-recurse-submodules' flag
	// More info: https://git-scm.com/docs/git-fetch#Documentation/git-fetch.txt---no-recurse-submodules
	fetchSubmodules bool
	// Sets `--filter=tree:0` flag
	// More info: https://github.blog/2020-12-21-get-up-to-speed-with-partial-clone-and-shallow-clone/#user-content-treeless-clones
	filterTree bool
}

// TODO
func (t fetchOptions) IsFullDepth() bool {
	return t.depth == 0
}

const (
	refsPrefix      = "refs/"
	refsHeadsPrefix = "refs/heads/"
)

func fetch(gitCmd git.Git, remote string, ref string, options fetchOptions) error {
	var opts []string
	opts = append(opts, jobsFlag)

	if options.limitDepth {
		opts = append(opts, fmt.Sprintf("--depth=%d", options.depth))
	}
	if options.filterTree {
		opts = append(opts, `--filter=tree:0`)
	}

	if options.tags {
		opts = append(opts, "--tags")
	} else {
		opts = append(opts, "--no-tags")
	}
	if !options.fetchSubmodules {
		opts = append(opts, "--no-recurse-submodules")
	}
	if ref != "" {
		opts = append(opts, remote, ref)
	}

	// Not necessarily a branch, can be tag too
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
			err,
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

		return fmt.Errorf("checkout failed (%s): %w", arg, cErr)
	}

	return nil
}

func fetchInitialBranch(gitCmd git.Git, remote string, branchRef string, fetchTraits fetchOptions) error {
	branch := strings.TrimPrefix(branchRef, refsHeadsPrefix)
	if err := fetch(gitCmd, remote, branchRef, fetchTraits); err != nil {
		wErr := fmt.Errorf("failed to fetch branch (%s): %w", branchRef, err)
		return fmt.Errorf("%v: %w", wErr, errors.New("please make sure the branch still exists"))
	}

	if err := checkoutWithCustomRetry(gitCmd, branch, nil); err != nil {
		return handleCheckoutError(
			listBranches(gitCmd),
			checkoutFailedTag,
			err,
			"Checkout has failed",
			branch,
		)
	}

	// Update branch: 'git fetch' followed by a 'git merge' is the same as 'git pull'.
	remoteBranch := fmt.Sprintf("%s/%s", remote, branch)
	if err := runner.Run(gitCmd.Merge(remoteBranch)); err != nil {
		return newStepError(
			"update_branch_failed",
			fmt.Errorf("updating branch (%s) failed: %w", branch, err),
			"Updating branch failed",
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

		wErr := fmt.Errorf("merge failed (%s): %w", arg, mErr)
		return fmt.Errorf("%v: %w", wErr, errors.New("please try to resolve all conflicts between the base and compare branches"))
	}

	return nil
}

func detachHead(gitCmd git.Git) error {
	if err := runner.Run(gitCmd.Checkout("--detach")); err != nil {
		return newStepError(
			"detach_head_failed",
			fmt.Errorf("detaching head failed: %w", err),
			"Detaching head failed",
		)
	}

	return nil
}
