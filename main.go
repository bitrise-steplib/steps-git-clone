package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/errorutil"
	"github.com/bitrise-io/go-utils/v2/exitcode"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-steplib/steps-git-clone/gitclone"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/tracker"
	"github.com/bitrise-steplib/steps-git-clone/step"
)

func main() {
	exitCode := run()
	os.Exit(int(exitCode))
}

func run() exitcode.ExitCode {
	logger := log.NewLogger()
	gitCloneStep := createStep(logger)

	cfg, err := gitCloneStep.ProcessConfig()
	if err != nil {
		logger.Println()
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to process Step inputs: %w", err))) //nolint:staticcheck
		return exitcode.Failure
	}

	result, err := gitCloneStep.Run(cfg)
	if err != nil {
		logger.Println()
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to execute Step: %w", err))) //nolint:staticcheck
		return exitcode.Failure
	}

	err = gitCloneStep.ExportOutputs(result)
	if err != nil {
		logger.Println()
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to export Step outputs: %w", err))) //nolint:staticcheck
	}

	fmt.Println()
	logger.Donef("Success")
	return exitcode.Success
}

func createStep(logger log.Logger) step.GitCloneStep {
	envRepo := env.NewRepository()
	stepTracker := tracker.NewStepTracker(envRepo, logger)
	inputParser := stepconf.NewInputParser(envRepo)
	cmdFactory := command.NewFactory(envRepo)
	pathChecker := pathutil.NewPathChecker()
	pathModifier := pathutil.NewPathModifier()
	runner := gitclone.NewDefaultRunner(logger)

	return step.NewGitCloneStep(step.GitCloneStepParams{
		Logger:       logger,
		Tracker:      stepTracker,
		InputParser:  inputParser,
		EnvRepo:      envRepo,
		CmdFactory:   cmdFactory,
		PathChecker:  pathChecker,
		PathModifier: pathModifier,
		Runner:       runner,
	})
}
