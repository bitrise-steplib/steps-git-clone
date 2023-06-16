package step

import (
	"fmt"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-git-clone/gitclone"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/bitriseapi"
	"github.com/bitrise-steplib/steps-git-clone/gitclone/tracker"
	"github.com/bitrise-steplib/steps-git-clone/transport"
)

type Input struct {
	ShouldMergePR bool `env:"merge_pr,opt[yes,no]"`

	GitHTTPUsername string `env:"git_http_username"`
	GitHTTPPassword string `env:"git_http_password"`

	CloneIntoDir         string   `env:"clone_into_dir,required"`
	CloneDepth           int      `env:"clone_depth"`
	UpdateSubmodules     bool     `env:"update_submodules,opt[yes,no]"`
	SubmoduleUpdateDepth int      `env:"submodule_update_depth"`
	FetchTags            bool     `env:"fetch_tags,opt[yes,no]"`
	SparseDirectories    []string `env:"sparse_directories,multiline"`

	RepositoryURL           string `env:"repository_url,required"`
	Commit                  string `env:"commit"`
	Tag                     string `env:"tag"`
	Branch                  string `env:"branch"`
	PRDestBranch            string `env:"branch_dest"`
	PRSourceRepositoryURL   string `env:"pull_request_repository_url"`
	PRMergeBranch           string `env:"pull_request_merge_branch"`
	PRUnverifiedMergeBranch string `env:"pull_request_unverified_merge_branch"`
	PRHeadBranch            string `env:"pull_request_head_branch"`

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
	tracker     tracker.StepTracker
	inputParser stepconf.InputParser
	cmdFactory  command.Factory
}

func NewGitCloneStep(logger log.Logger, tracker tracker.StepTracker, inputParser stepconf.InputParser, cmdFactory command.Factory) GitCloneStep {
	return GitCloneStep{
		logger:      logger,
		tracker:     tracker,
		inputParser: inputParser,
		cmdFactory:  cmdFactory,
	}
}

func (g GitCloneStep) ProcessConfig() (Config, error) {
	var input Input
	if err := g.inputParser.Parse(&input); err != nil {
		return Config{}, fmt.Errorf("Error: %s\n", err)
	}
	stepconf.Print(input)

	return Config{input}, nil
}

func (g GitCloneStep) Run(cfg Config) (gitclone.CheckoutStateResult, error) {
	if err := transport.Setup(transport.Config{URL: cfg.RepositoryURL, HTTPUsername: cfg.GitHTTPUsername, HTTPPassword: cfg.GitHTTPPassword}); err != nil {
		return gitclone.CheckoutStateResult{}, err
	}
	gitCloneCfg := convertConfig(cfg)
	patchSource := bitriseapi.NewPatchSource(cfg.BuildURL, cfg.BuildAPIToken)
	mergeRefChecker := bitriseapi.NewMergeRefChecker(cfg.BuildURL, cfg.BuildAPIToken, retry.NewHTTPClient(), g.logger, g.tracker)
	cloner := gitclone.NewGitCloner(g.logger, g.tracker, g.cmdFactory, patchSource, mergeRefChecker)
	return cloner.CheckoutState(gitCloneCfg)
}

func (g GitCloneStep) ExportOutputs(runResult gitclone.CheckoutStateResult) error {
	fmt.Println()

	exporter := gitclone.NewOutputExporter(g.logger, g.cmdFactory, runResult)
	if err := exporter.ExportCommitInfo(); err != nil {
		return err
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
		PRMergeRef:            config.PRMergeBranch,
		PRUnverifiedMergeRef:  config.PRUnverifiedMergeBranch,
		PRHeadBranch:          config.PRHeadBranch,
		ResetRepository:       config.ResetRepository,
	}
}
