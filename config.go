package main

import (
	"errors"
	"os"

	"github.com/bitrise-io/go-utils/log"
)

// Config ...
type Config struct {
	CloneIntoDir  string
	RepositoryURL string
	Commit        string
	Tag           string
	Branch        string
	CloneDepth    string

	PRRepositoryCloneURL string
	PRID                 string
	BranchDest           string
	PRMergeBranch        string
	resetRepository      string

	BuildURL         string
	BuildAPIToken    string
	updateSubmodules string
	manualMerge      string
}

func newConfig() Config {
	return Config{
		CloneIntoDir:  os.Getenv("clone_into_dir"),
		RepositoryURL: os.Getenv("repository_url"),
		Commit:        os.Getenv("commit"),
		Tag:           os.Getenv("tag"),
		Branch:        os.Getenv("branch"),
		CloneDepth:    os.Getenv("clone_depth"),

		PRRepositoryCloneURL: os.Getenv("pull_request_repository_url"),
		PRID:                 os.Getenv("pull_request_id"),
		BranchDest:           os.Getenv("branch_dest"),
		PRMergeBranch:        os.Getenv("pull_request_merge_branch"),
		resetRepository:      os.Getenv("reset_repository"),
		manualMerge:          os.Getenv("manual_merge"),

		BuildURL:         os.Getenv("build_url"),
		BuildAPIToken:    os.Getenv("build_api_token"),
		updateSubmodules: os.Getenv("update_submodules"),
	}
}

func (c Config) print() {
	log.Infof("Git Clone config:")
	log.Printf("- CloneIntoDir: %s", c.CloneIntoDir)
	log.Printf("- RepositoryURL: %s", c.RepositoryURL)
	log.Printf("- UpdateSubmodules: %s", c.updateSubmodules)

	log.Infof("Git Checkout config:")
	log.Printf("- Commit: %s", c.Commit)
	log.Printf("- Tag: %s", c.Tag)
	log.Printf("- Branch: %s", c.Branch)
	log.Printf("- CloneDepth: %s", c.CloneDepth)

	log.Infof("Git Pull Request config:")
	log.Printf("- PRRepositoryCloneURL: %s", c.PRRepositoryCloneURL)
	log.Printf("- PRID: %s", c.PRID)
	log.Printf("- BranchDest: %s", c.BranchDest)
	log.Printf("- PRMergeBranch: %s", c.PRMergeBranch)
	log.Printf("- ResetRepository: %s", c.resetRepository)
	log.Printf("- ManualMerge: %s", c.manualMerge)

	log.Infof("Bitrise Build config:")
	log.Printf("- BuildURL: %s", c.BuildURL)
	log.Printf("- BuildAPIToken: %s", c.BuildAPIToken)
}

func (c Config) validate() error {
	if c.CloneIntoDir == "" {
		return errors.New("no CloneIntoDir parameter specified")
	}
	if c.RepositoryURL == "" {
		return errors.New("no RepositoryURL parameter specified")
	}
	return nil
}

// ResetRepository ...
func (c Config) ResetRepository() bool {
	return c.resetRepository == "yes"
}

// ManualMerge ...
func (c Config) ManualMerge() bool {
	return c.manualMerge == "yes"
}

// UpdateSubmodules ...
func (c Config) UpdateSubmodules() bool {
	return c.updateSubmodules == "yes"
}
