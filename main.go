package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/steps-git-clone/analytics"
	"github.com/bitrise-io/steps-git-clone/gitutil"
)

const (
	retryCount = 2
	waitTime   = 5 //seconds
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

	BuildURL      string
	BuildAPIToken string
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

		BuildURL:      os.Getenv("build_url"),
		BuildAPIToken: os.Getenv("build_api_token"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Git Clone Configs:")
	log.Printf("- CloneIntoDir: %s", configs.CloneIntoDir)
	log.Printf("- RepositoryURL: %s", configs.RepositoryURL)

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

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := command.New("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

// -----------------------
// --- Main
// -----------------------
func main() {
	//
	// Validate options
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		analytics.LogError("parse_config_failed", err, "Parsing configuration has failed")
		log.Errorf("Issue with input: %s", err)
		os.Exit(1)
	}
	fmt.Println()
	// ---

	// git
	log.Infof("Git clone repository")

	git, err := gitutil.NewHelper(configs.CloneIntoDir, configs.RepositoryURL, configs.ResetRepository == "Yes")
	if err != nil {
		analytics.LogError("construct_git_helper_failed", err, "Constructing new git helper has failed")
		log.Errorf("Failed to create git helper, error: %s", err)
		os.Exit(1)
	}

	git.ConfigureCheckout(configs.PullRequestID, configs.PullRequestURI, configs.PullRequestMergeBranch, configs.Commit, configs.Tag, configs.Branch, configs.BranchDest, configs.CloneDepth, configs.BuildURL, configs.BuildAPIToken)

	if err := git.Init(); err != nil {
		log.Errorf("Failed, error: %s", err)
		analytics.LogError("init_git_failed", err, "Initializing git has failed")
		os.Exit(1)
	}

	if !git.IsOriginPresented() {
		if err := git.RemoteAdd(); err != nil {
			analytics.LogError("add_remote_failed", err, "Adding remote to git has failed")
			log.Errorf("Failed, error: %s", err)
			os.Exit(1)
		}
	}

	if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("Retrying...")
		}

		fetchErr := git.Fetch()
		if fetchErr != nil {
			log.Warnf("%d attempt failed:", attempt)
			fmt.Println(fetchErr.Error())
		}

		return fetchErr
	}); err != nil {
		analytics.LogError("fetch_failed", err, "Fetching repository has failed")
		log.Errorf("Failed, error: %s", err)
		os.Exit(1)
	}

	if git.ShouldCheckout() {
		if git.ShouldCheckoutTag() {
			if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
				if attempt > 0 {
					log.Warnf("Retrying...")
				}

				fetchErr := git.FetchTags()
				if fetchErr != nil {
					log.Warnf("%d attempt failed:", attempt)
					fmt.Println(fetchErr.Error())
				}

				return fetchErr
			}); err != nil {
				analytics.LogError("fetch_tags_failed", err, "Fetching tags has failed")
				log.Errorf("Failed, error: %s", err)
				os.Exit(1)
			}
		}

		if err := retry.Times(1).Wait(waitTime).Try(func(attempt uint) error {
			if attempt > 0 {
				log.Warnf("Retry with fetching tags...")
				fetchErr := git.FetchTags()
				if fetchErr != nil {
					log.Warnf("Fetch tags attempt failed")
					fmt.Println(fetchErr.Error())
				}
			}
			checkoutErr := git.Checkout()
			if checkoutErr != nil {
				log.Errorf("Checkout failed, error: %s", checkoutErr)
			}
			return checkoutErr
		}); err != nil {
			if !git.ShouldTryFetchUnshallow() {
				analytics.LogError("checkout_failed", err, "Checkout has failed")
				log.Errorf("Failed, error: %s", err)
				os.Exit(1)
			}

			log.Warnf("Failed, error: %s", err)
			log.Warnf("Unshallow...")

			if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
				if attempt > 0 {
					log.Warnf("Retrying...")
				}

				fetchShallowErr := git.FetchUnshallow()
				if fetchShallowErr != nil {
					log.Warnf("%d attempt failed:", attempt)
					fmt.Println(fetchShallowErr.Error())
				}

				return fetchShallowErr
			}); err != nil {
				analytics.LogError("fetch_unshallow_failed", err, "Fetching with unshallow parameter has failed")
				log.Errorf("Failed, error: %s", err)
				os.Exit(1)
			}

			if err := git.Checkout(); err != nil {
				analytics.LogError("checkout_unshallow_failed", err, "Checkout after unshallow fetch has failed")
				log.Errorf("Failed, error: %s", err)
				os.Exit(1)
			}
		}

		if git.ShouldMergePullRequest() {
			if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
				if attempt > 0 {
					log.Warnf("Retrying...")
				}

				gitMergeErr := git.MergePullRequest()
				if gitMergeErr != nil {
					log.Warnf("%d attempt failed:", attempt)
					fmt.Println(gitMergeErr.Error())
				}

				return gitMergeErr
			}); err != nil {
				analytics.LogError("merge_failed", err, "Merging pr has failed")
				log.Errorf("Failed, error: %s", err)
				os.Exit(1)
			}
		}

		if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
			if attempt > 0 {
				log.Warnf("Retrying...")
			}

			submoduleErr := git.SubmoduleUpdate()
			if submoduleErr != nil {
				log.Warnf("%d attempt failed:", attempt)
				fmt.Println(submoduleErr.Error())
			}

			return submoduleErr
		}); err != nil {
			analytics.LogError("update_submodule_failed", err, "Updating submodule has failed")
			log.Errorf("Failed, error: %s", err)
			os.Exit(1)
		}

		log.Infof("Exporting git logs")

		if commitHash, err := git.LogCommitHash(); err != nil {
			analytics.LogError("fetch_commit_hash_failed", err, "Obtaining commit hash has failed")
			log.Errorf("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Printf("=> GIT_CLONE_COMMIT_HASH")
			log.Printf("   value: %s", commitHash)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_HASH", commitHash); err != nil {
				log.Warnf("envman export failed, error: %s", err)
			}
		}

		if commitMessageSubject, err := git.LogCommitMessageSubject(); err != nil {
			analytics.LogError("fetch_commit_message_subject_failed", err, "Obtaining commit message's subject has failed")
			log.Errorf("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Printf("=> GIT_CLONE_COMMIT_MESSAGE_SUBJECT")
			log.Printf("   value: %s", commitMessageSubject)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_MESSAGE_SUBJECT", commitMessageSubject); err != nil {
				log.Warnf("envman export failed, error: %s", err)
			}
		}

		if commitMessageBody, err := git.LogCommitMessageBody(); err != nil {
			analytics.LogError("fetch_commit_message_body_failed", err, "Obtaining commit message's body has failed")
			log.Errorf("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Printf("=> GIT_CLONE_COMMIT_MESSAGE_BODY")
			log.Printf("   value: %s", commitMessageBody)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_MESSAGE_BODY", commitMessageBody); err != nil {
				log.Warnf("envman export failed, error: %s", err)
			}
		}

		if commitAuthorName, err := git.LogAuthorName(); err != nil {
			analytics.LogError("fetch_commit_author_name_failed", err, "Obtaining commit author's name has failed")
			log.Errorf("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Printf("=> GIT_CLONE_COMMIT_AUTHOR_NAME")
			log.Printf("   value: %s", commitAuthorName)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_AUTHOR_NAME", commitAuthorName); err != nil {
				log.Warnf("envman export failed, error: %s", err)
			}
		}

		if commitAuthorEmail, err := git.LogAuthorEmail(); err != nil {
			analytics.LogError("fetch_commit_author_email_failed", err, "Obtaining commit author's e-mail has failed")
			log.Errorf("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Printf("=> GIT_CLONE_COMMIT_AUTHOR_EMAIL")
			log.Printf("   value: %s", commitAuthorEmail)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_AUTHOR_EMAIL", commitAuthorEmail); err != nil {
				log.Warnf("envman export failed, error: %s", err)
			}
		}

		if commitCommiterName, err := git.LogCommiterName(); err != nil {
			analytics.LogError("fetch_commit_commiter_name_failed", err, "Obtaining commit commiter's name has failed")
			log.Errorf("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Printf("=> GIT_CLONE_COMMIT_COMMITER_NAME")
			log.Printf("   value: %s", commitCommiterName)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COMMITER_NAME", commitCommiterName); err != nil {
				log.Warnf("envman export failed, error: %s", err)
			}
		}

		if commitCommiterEmail, err := git.LogCommiterEmail(); err != nil {
			analytics.LogError("fetch_commit_commiter_email_failed", err, "Obtaining commit commiter's email has failed")
			log.Errorf("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Printf("=> GIT_CLONE_COMMIT_COMMITER_EMAIL")
			log.Printf("   value: %s", commitCommiterEmail)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COMMITER_EMAIL", commitCommiterEmail); err != nil {
				log.Warnf("envman export failed, error: %s", err)
			}
		}
	}

	log.Donef("Success")
}
