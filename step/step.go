package step

import (
	"fmt"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-git-clone/gitclone"
)

type Input struct {
	ShouldMergePR bool `env:"merge_pr,opt[yes,no]"`

	CloneIntoDir         string   `env:"clone_into_dir,required"`
	CloneDepth           int      `env:"clone_depth"`
	UpdateSubmodules     bool     `env:"update_submodules,opt[yes,no]"`
	SubmoduleUpdateDepth int      `env:"submodule_update_depth"`
	FetchTags            bool     `env:"fetch_tags,opt[yes,no]"`
	SparseDirectories    []string `env:"sparse_directories,multiline"`

	RepositoryURL         string `env:"repository_url,required"`
	Commit                string `env:"commit"`
	Tag                   string `env:"tag"`
	Branch                string `env:"branch"`
	PRDestBranch          string `env:"branch_dest"`
	PRSourceRepositoryURL string `env:"pull_request_repository_url"`
	PRMergeBranch         string `env:"pull_request_merge_branch"`
	PRHeadBranch          string `env:"pull_request_head_branch"`

	ResetRepository bool   `env:"reset_repository,opt[Yes,No]"`
	BuildURL        string `env:"build_url"`
	BuildAPIToken   string `env:"build_api_token"`
}

// Config is the git clone step configuration
type Config struct {
	Input
}

type GitCloneStep struct {
	logger      log.Logger
	tracker     gitclone.StepTracker
	inputParser stepconf.InputParser
	cmdFactory  command.Factory
}

func NewGitCloneStep(logger log.Logger, tracker gitclone.StepTracker, inputParser stepconf.InputParser, cmdFactory command.Factory) GitCloneStep {
	return GitCloneStep{
		logger:      logger,
		tracker:     tracker,
		inputParser: inputParser,
		cmdFactory:  cmdFactory,
	}
}

func (g GitCloneStep) ProcessConfig() (Config, error) {
	var cfg Config
	if err := g.inputParser.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("Error: %s\n", err)
	}
	stepconf.Print(cfg)

	return cfg, nil
}

func (g GitCloneStep) Execute(cfg Config) (gitclone.CheckoutStateResult, error) {
	gitcloneCfg := convertConfig(cfg)
	cloner := gitclone.NewGitCloner(g.logger, g.tracker, g.cmdFactory)
	return cloner.CheckoutState(gitcloneCfg)
}

func (g GitCloneStep) ExportOutputs(runResult gitclone.CheckoutStateResult) error {
	fmt.Println()
	g.logger.Infof("Exporting commit details")
	ref := runResult.CheckoutStrategy.GetBuildTriggerRef()
	if ref == "" {
		g.logger.Warnf(`Can't export commit information (commit message and author) as it is not available.
This is a limitation of Bitbucket webhooks when the PR source repo (a fork) is not accessible.
Try using the env vars based on the webhook contents instead, such as $BITRISE_GIT_COMMIT and $BITRISE_GIT_MESSAGE`)
		return nil
	}

	exporter := gitclone.NewOutputExporter(g.logger, g.cmdFactory, runResult.GitCmd)
	if err := exporter.ExportCommitInfo(ref, runResult.IsPR); err != nil {
		return gitclone.NewStepError("export_envs_failed", err, "Exporting envs failed")
	}

	return nil
}

func convertConfig(config Config) gitclone.Config {
	return gitclone.Config{
		ShouldMergePR:         config.ShouldMergePR,
		CloneIntoDir:          config.CloneIntoDir,
		CloneDepth:            config.CloneDepth,
		UpdateSubmodules:      config.UpdateSubmodules,
		SubmoduleUpdateDepth:  config.SubmoduleUpdateDepth,
		FetchTags:             config.FetchTags,
		SparseDirectories:     config.SparseDirectories,
		RepositoryURL:         config.RepositoryURL,
		Commit:                config.Commit,
		Tag:                   config.Tag,
		Branch:                config.Branch,
		PRDestBranch:          config.PRDestBranch,
		PRSourceRepositoryURL: config.PRSourceRepositoryURL,
		PRMergeBranch:         config.PRMergeBranch,
		PRHeadBranch:          config.PRHeadBranch,
		ResetRepository:       config.ResetRepository,
		BuildURL:              config.BuildURL,
		BuildAPIToken:         config.BuildAPIToken,
	}
}
