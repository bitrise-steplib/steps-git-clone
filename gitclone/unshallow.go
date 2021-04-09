package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

type unshallowFetchOptions struct {
	// Sets '--tags' or `--no-tags` flag
	// More info:
	// - https://git-scm.com/docs/git-fetch#Documentation/git-fetch.txt---tags
	// - https://git-scm.com/docs/git-fetch#Documentation/git-fetch.txt---no-tags
	tags bool
	// Sets '--no-recurse-submodules' flag
	// More info: https://git-scm.com/docs/git-fetch#Documentation/git-fetch.txt---no-recurse-submodules
	fetchSubmodules bool
}

type fallbackRetry interface {
	do(gitCmd git.Git) error
}

type simpleUnshallow struct {
	traits unshallowFetchOptions
}

func (s simpleUnshallow) do(gitCmd git.Git) error {
	log.Infof("Fetch with unshallow...")

	return unshallowFetch(gitCmd, s.traits)
}

type resetUnshallow struct {
	traits unshallowFetchOptions
}

func (r resetUnshallow) do(gitCmd git.Git) error {
	log.Infof("Resetting repository, then fetch with unshallow...")

	if err := resetRepo(gitCmd); err != nil {
		return fmt.Errorf("reset repository: %v", err)
	}

	return unshallowFetch(gitCmd, r.traits)
}

func unshallowFetch(gitCmd git.Git, traits unshallowFetchOptions) error {
	opts := []string{jobsFlag, "--unshallow"}
	if traits.tags {
		opts = append(opts, "--tags")
	} else {
		opts = append(opts, "--no-tags")
	}
	if !traits.fetchSubmodules {
		opts = append(opts, "--no-recurse-submodules")
	}

	if err := runner.RunWithRetry(func() *command.Model {
		return gitCmd.Fetch(opts...)
	}); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}
	return nil
}
