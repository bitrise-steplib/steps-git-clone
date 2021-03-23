package gitclone

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

type fallbackRetry interface {
	do(gitCmd git.Git) error
}

type simpleUnshallow struct{}

func (s simpleUnshallow) do(gitCmd git.Git) error {
	log.Infof("Fetch with unshallow...")

	if err := runner.RunWithRetry(func() *command.Model {
		return gitCmd.Fetch("--unshallow")
	}); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}

	return nil
}

type resetUnshallow struct{}

func (r resetUnshallow) do(gitCmd git.Git) error {
	log.Infof("Resetting repository, then fetch with unshallow...")

	if err := resetRepo(gitCmd); err != nil {
		return fmt.Errorf("reset repository: %v", err)
	}
	if err := runner.RunWithRetry(func() *command.Model {
		return gitCmd.Fetch("--unshallow")
	}); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}

	return nil
}
