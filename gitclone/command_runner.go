package gitclone

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
)

// CommandRunner ...
type CommandRunner interface {
	RunForOutput(c *command.Model) (string, error)
	Run(c *command.Model) error
	RunWithRetry(getCommmand func() *command.Model) error
}

// DefaultRunner ...
type DefaultRunner struct {
}

// RunForOutput ...
func (r DefaultRunner) RunForOutput(c *command.Model) (string, error) {
	log.Infof("$ %s &> out", c.PrintableCommandArgs())

	out, err := c.RunAndReturnTrimmedCombinedOutput()
	if err != nil && errorutil.IsExitStatusError(err) {
		return out, errors.New(out)
	}

	return out, err
}

// Run ...
func (r DefaultRunner) Run(c *command.Model) error {
	fmt.Println()
	log.Infof("$ %s", c.PrintableCommandArgs())
	var buffer bytes.Buffer

	err := c.SetStdout(os.Stdout).SetStderr(io.MultiWriter(os.Stderr, &buffer)).Run()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return errors.New(strings.TrimSpace(buffer.String()))
		}
		return err
	}

	return nil
}

// RunWithRetry ...
func (r DefaultRunner) RunWithRetry(getCommand func() *command.Model) error {
	return retry.Times(2).Wait(5).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("Retrying...")
		}

		err := r.Run(getCommand())
		if err != nil {
			log.Warnf("Attempt %d failed:", attempt+1)
			fmt.Println(err.Error())
		}

		return err
	})
}
