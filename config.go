package main

import (
	"errors"
	"os"

	"github.com/bitrise-io/go-utils/log"
)

// ConfigsModel ...
type ConfigsModel struct {
	CloneIntoDir  string
	RepositoryURL string
	Commit        string
	Tag           string
	Branch        string
	CloneDepth    string

	PullRequestURI         string
	PullRequestID          string
	BranchDest             string
	PullRequestMergeBranch string
	ResetRepository        string

	BuildURL         string
	BuildAPIToken    string
	UpdateSubmodules string
	ManualMerge      string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		CloneIntoDir:  os.Getenv("clone_into_dir"),
		RepositoryURL: os.Getenv("repository_url"),
		Commit:        os.Getenv("commit"),
		Tag:           os.Getenv("tag"),
		Branch:        os.Getenv("branch"),
		CloneDepth:    os.Getenv("clone_depth"),

		PullRequestURI:         os.Getenv("pull_request_repository_url"),
		PullRequestID:          os.Getenv("pull_request_id"),
		BranchDest:             os.Getenv("branch_dest"),
		PullRequestMergeBranch: os.Getenv("pull_request_merge_branch"),
		ResetRepository:        os.Getenv("reset_repository"),
		ManualMerge:            os.Getenv("manual_merge"),

		BuildURL:         os.Getenv("build_url"),
		BuildAPIToken:    os.Getenv("build_api_token"),
		UpdateSubmodules: os.Getenv("update_submodules"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Git Clone Configs:")
	log.Printf("- CloneIntoDir: %s", configs.CloneIntoDir)
	log.Printf("- RepositoryURL: %s", configs.RepositoryURL)
	log.Printf("- UpdateSubmodules: %s", configs.UpdateSubmodules)

	log.Infof("Git Checkout Configs:")
	log.Printf("- Commit: %s", configs.Commit)
	log.Printf("- Tag: %s", configs.Tag)
	log.Printf("- Branch: %s", configs.Branch)
	log.Printf("- CloneDepth: %s", configs.CloneDepth)

	log.Infof("Git Pull Request Configs:")
	log.Printf("- PullRequestURI: %s", configs.PullRequestURI)
	log.Printf("- PullRequestID: %s", configs.PullRequestID)
	log.Printf("- BranchDest: %s", configs.BranchDest)
	log.Printf("- PullRequestMergeBranch: %s", configs.PullRequestMergeBranch)
	log.Printf("- ResetRepository: %s", configs.ResetRepository)
	log.Printf("- ManualMerge: %s", configs.ManualMerge)

	log.Infof("Bitrise Build Configs:")
	log.Printf("- BuildURL: %s", configs.BuildURL)
	log.Printf("- BuildAPIToken: %s", configs.BuildAPIToken)
}

func (configs ConfigsModel) validate() error {
	if configs.CloneIntoDir == "" {
		return errors.New("no CloneIntoDir parameter specified")
	}
	if configs.RepositoryURL == "" {
		return errors.New("no RepositoryURL parameter specified")
	}
	return nil
}
