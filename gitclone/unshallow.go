package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/v2/git"
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
	do(cloner *GitCloner, gitFactory git.Factory) error
}

type simpleUnshallow struct {
	traits unshallowFetchOptions
}

func (s simpleUnshallow) do(cloner *GitCloner, gitFactory git.Factory) error {
	cloner.logger.Infof("Fetch with unshallow...")

	return cloner.unshallowFetch(gitFactory, s.traits)
}

type resetUnshallow struct {
	traits unshallowFetchOptions
}

func (r resetUnshallow) do(cloner *GitCloner, gitFactory git.Factory) error {
	cloner.logger.Infof("Resetting repository, then fetch with unshallow...")

	if err := cloner.resetRepo(gitFactory); err != nil {
		return fmt.Errorf("reset repository: %v", err)
	}

	return cloner.unshallowFetch(gitFactory, r.traits)
}

func (g GitCloner) unshallowFetch(gitFactory git.Factory, traits unshallowFetchOptions) error {
	opts := []string{jobsFlag, "--unshallow"}
	if traits.tags {
		opts = append(opts, "--tags")
	} else {
		opts = append(opts, "--no-tags")
	}
	if !traits.fetchSubmodules {
		opts = append(opts, "--no-recurse-submodules")
	}

	if err := g.runner.RunWithRetry(func() git.Template {
		return gitFactory.Fetch(opts...)
	}); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}
	return nil
}
