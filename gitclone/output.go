package gitclone

import (
	"fmt"

	"github.com/bitrise-io/envman/envman"
	"github.com/bitrise-io/go-steputils/v2/export"
	v1command "github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
)

const outputCommitterName = "GIT_CLONE_COMMIT_COMMITTER_NAME"
const outputCommitterEmail = "GIT_CLONE_COMMIT_COMMITTER_EMAIL"
const outputCommitCount = "GIT_CLONE_COMMIT_COUNT"

type gitOutput struct {
	envKey string
	gitCmd *v1command.Model
}

type OutputExporter struct {
	logger   log.Logger
	gitCmd   git.Git
	exporter export.Exporter
}

func NewOutputExporter(logger log.Logger, cmdFactory command.Factory, gitCmd git.Git) OutputExporter {
	return OutputExporter{
		logger:   logger,
		gitCmd:   gitCmd,
		exporter: export.NewExporter(cmdFactory),
	}
}

func (e *OutputExporter) ExportCommitInfo(gitRef string, isPR bool) error {
	maxEnvLength, err := getMaxEnvLength()
	if err != nil {
		return err
	}

	for _, commitInfo := range e.gitOutputs(gitRef, isPR) {
		if err := e.printLogAndExportEnv(commitInfo.gitCmd, commitInfo.envKey, maxEnvLength); err != nil {
			return err
		}
	}

	return nil
}

func (e *OutputExporter) gitOutputs(gitRef string, isPR bool) []gitOutput {
	outputs := []gitOutput{
		{
			envKey: "GIT_CLONE_COMMIT_AUTHOR_NAME",
			gitCmd: e.gitCmd.Log(`%an`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
			gitCmd: e.gitCmd.Log(`%ae`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_HASH",
			gitCmd: e.gitCmd.Log(`%H`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
			gitCmd: e.gitCmd.Log(`%s`, gitRef),
		},
		{
			envKey: "GIT_CLONE_COMMIT_MESSAGE_BODY",
			gitCmd: e.gitCmd.Log(`%b`, gitRef),
		},
	}
	if isPR {
		e.logger.Printf("The following outputs are not exported for Pull Requests:")
		e.logger.Printf("- %s", outputCommitterName)
		e.logger.Printf("- %s", outputCommitterEmail)
		e.logger.Printf("- %s", outputCommitCount)
	} else {
		extraOutputs := []gitOutput{
			{
				envKey: outputCommitterName,
				gitCmd: e.gitCmd.Log(`%cn`, gitRef),
			},
			{
				envKey: outputCommitterEmail,
				gitCmd: e.gitCmd.Log(`%ce`, gitRef),
			},
			{
				envKey: outputCommitCount,
				gitCmd: e.gitCmd.RevList("HEAD", "--count"),
			},
		}
		outputs = append(outputs, extraOutputs...)
	}

	return outputs
}

func (e *OutputExporter) printLogAndExportEnv(command *v1command.Model, env string, maxEnvLength int) error {
	l, err := runner.RunForOutput(command)
	if err != nil {
		return fmt.Errorf("command failed: %s", err)
	}

	if (env == "GIT_CLONE_COMMIT_MESSAGE_SUBJECT" || env == "GIT_CLONE_COMMIT_MESSAGE_BODY") && len(l) > maxEnvLength {
		tv := l[:maxEnvLength-len(trimEnding)] + trimEnding
		e.logger.Printf("Value %s  is bigger than maximum env variable size, trimming", env)
		l = tv
	}

	e.logger.Printf("=> %s\n   value: %s", env, l)
	if err := e.exporter.ExportOutput(env, l); err != nil {
		return fmt.Errorf("envman export failed: %v", err)
	}
	return nil
}

func getMaxEnvLength() (int, error) {
	configs, err := envman.GetConfigs()
	if err != nil {
		return 0, err
	}

	return configs.EnvBytesLimitInKB * 1024, nil
}
