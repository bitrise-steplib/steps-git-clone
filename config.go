package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/input"
)

// Config ...
type Config struct {
	CloneIntoDir  string
	RepositoryURL string
	Commit        string
	Tag           string
	Branch        string
	CloneDepth    int

	PRRepositoryCloneURL string
	PRID                 int
	BranchDest           string
	PRMergeBranch        string
	ResetRepository      bool

	BuildURL         string
	BuildAPIToken    string
	UpdateSubmodules bool
	ManualMerge      bool
}

func newConfig() (Config, []error) {
	errs := []error{}
	config := Config{
		Commit: os.Getenv("commit"),
		Tag:    os.Getenv("tag"),
		Branch: os.Getenv("branch"),

		PRRepositoryCloneURL: os.Getenv("pull_request_repository_url"),
		BranchDest:           os.Getenv("branch_dest"),
		PRMergeBranch:        os.Getenv("pull_request_merge_branch"),

		BuildURL:      os.Getenv("build_url"),
		BuildAPIToken: os.Getenv("build_api_token"),
	}

	// required
	err := input.ValidateIfNotEmpty(os.Getenv("clone_into_dir"))
	if err != nil {
		errs = append(errs, fmt.Errorf("clone_into_dir: %v", err))
	} else {
		config.CloneIntoDir = os.Getenv("clone_into_dir")
	}

	err = input.ValidateIfNotEmpty(os.Getenv("repository_url"))
	if err != nil {
		errs = append(errs, fmt.Errorf("repository_url: %v", err))
	} else {
		config.RepositoryURL = os.Getenv("repository_url")
	}

	// numbers
	num, err := input.ValidateInt(os.Getenv("clone_depth"))
	if err != nil {
		errs = append(errs, fmt.Errorf("clone_depth: %v", err))
	} else {
		config.CloneDepth = num
	}

	num, err = input.ValidateInt(os.Getenv("pull_request_id"))
	if err != nil {
		errs = append(errs, fmt.Errorf("pull_request_id: %v", err))
	} else {
		config.PRID = num
	}

	// bools
	err = input.ValidateWithOptions(os.Getenv("reset_repository"), "yes", "no")
	if err != nil {
		errs = append(errs, fmt.Errorf("reset_repository: %v", err))
	} else {
		config.ResetRepository = os.Getenv("reset_repository") == "yes"
	}

	err = input.ValidateWithOptions(os.Getenv("manual_merge"), "yes", "no")
	if err != nil {
		errs = append(errs, fmt.Errorf("manual_merge: %v", err))
	} else {
		config.ManualMerge = os.Getenv("manual_merge") == "yes"
	}

	err = input.ValidateWithOptions(os.Getenv("update_submodules"), "yes", "no")
	if err != nil {
		errs = append(errs, fmt.Errorf("update_submodules: %v", err))
	} else {
		config.UpdateSubmodules = os.Getenv("update_submodules") == "yes"
	}

	return config, errs
}

func (c Config) print() {
	log.Infof("Git Clone config:")
	log.Printf("- CloneIntoDir: %s", c.CloneIntoDir)
	log.Printf("- RepositoryURL: %s", c.RepositoryURL)
	log.Printf("- UpdateSubmodules: %t", c.UpdateSubmodules)

	log.Infof("Git Checkout config:")
	log.Printf("- Commit: %s", c.Commit)
	log.Printf("- Tag: %s", c.Tag)
	log.Printf("- Branch: %s", c.Branch)
	log.Printf("- CloneDepth: %d", c.CloneDepth)

	log.Infof("Git Pull Request config:")
	log.Printf("- PRRepositoryCloneURL: %s", c.PRRepositoryCloneURL)
	log.Printf("- PRID: %d", c.PRID)
	log.Printf("- BranchDest: %s", c.BranchDest)
	log.Printf("- PRMergeBranch: %s", c.PRMergeBranch)
	log.Printf("- ResetRepository: %t", c.ResetRepository)
	log.Printf("- ManualMerge: %t", c.ManualMerge)

	log.Infof("Bitrise Build config:")
	log.Printf("- BuildURL: %s", c.BuildURL)
	log.Printf("- BuildAPIToken: %s", c.BuildAPIToken)
	fmt.Println()
}
