package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/errorutil"
	. "github.com/bitrise-io/go-utils/v2/exitcode"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-git-clone/gitclone"
	"github.com/bitrise-steplib/steps-git-clone/step"
)

func main() {
	exitCode := run()
	os.Exit(int(exitCode))
}

func run() ExitCode {
	logger := log.NewLogger()
	gitCloneStep := createStep(logger)

	cfg, err := gitCloneStep.ProcessConfig()
	if err != nil {
		logger.Println()
		logger.Errorf(errorutil.FormattedError(fmt.Errorf("Failed to process Step inputs: %w", err)))
		return Failure
	}

	result, err := gitCloneStep.Run(cfg)
	if err != nil {
		logger.Println()
		logger.Errorf(errorutil.FormattedError(fmt.Errorf("Failed to execute Step: %w", err)))
		return Failure
	}

	err = gitCloneStep.ExportOutputs(result)
	if err != nil {
		logger.Println()
		logger.Errorf(errorutil.FormattedError(fmt.Errorf("Failed to export Step outputs: %w", err)))
	}

	fmt.Println()
	logger.Donef("Success")
	return Success
}

func createStep(logger log.Logger) step.GitCloneStep {
	envRepo := env.NewRepository()
	tracker := gitclone.NewStepTracker(envRepo, logger)
	inputParser := stepconf.NewInputParser(envRepo)
	cmdFactory := command.NewFactory(envRepo)

	return step.NewGitCloneStep(logger, tracker, inputParser, cmdFactory)
}
