package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
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
	log.Info("Git Clone Configs:")
	log.Detail("- CloneIntoDir: %s", configs.CloneIntoDir)
	log.Detail("- RepositoryURL: %s", configs.RepositoryURL)

	log.Info("Git Checkout Configs:")
	log.Detail("- Commit: %s", configs.Commit)
	log.Detail("- Tag: %s", configs.Tag)
	log.Detail("- Branch: %s", configs.Branch)
	log.Detail("- CloneDepth: %s", configs.CloneDepth)

	log.Info("Git Pull Request Configs:")
	log.Detail("- PullRequestURI: %s", configs.PullRequestURI)
	log.Detail("- PullRequestID: %s", configs.PullRequestID)
	log.Detail("- BranchDest: %s", configs.BranchDest)
	log.Detail("- PullRequestMergeBranch: %s", configs.PullRequestMergeBranch)
	log.Detail("- ResetRepository: %s", configs.ResetRepository)

	log.Info("Bitrise Build Configs:")
	log.Detail("- BuildURL: %s", configs.BuildURL)
	log.Detail("- BuildAPIToken: %s", configs.BuildAPIToken)
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
	cmd := cmdex.NewCommand("envman", "add", "--key", keyStr)
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
		log.Error("Issue with input: %s", err)
		os.Exit(1)
	}
	fmt.Println()
	// ---

	// git
	log.Info("Git clone repository")

	git, err := gitutil.NewHelper(configs.CloneIntoDir, configs.RepositoryURL, configs.ResetRepository == "Yes")
	if err != nil {
		log.Error("Failed to create git helper, error: %s", err)
		os.Exit(1)
	}

	git.ConfigureCheckout(configs.PullRequestID, configs.PullRequestURI, configs.PullRequestMergeBranch, configs.Commit, configs.Tag, configs.Branch, configs.BranchDest, configs.CloneDepth, configs.BuildURL, configs.BuildAPIToken)

	if err := git.Init(); err != nil {
		log.Error("Failed, error: %s", err)
		os.Exit(1)
	}

	if !git.IsOriginPresented() {
		if err := git.RemoteAdd(); err != nil {
			log.Error("Failed, error: %s", err)
			os.Exit(1)
		}
	}

	if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warn("Retrying...")
		}

		fetchErr := git.Fetch()
		if fetchErr != nil {
			log.Warn("%d attempt failed:", attempt)
			fmt.Println(fetchErr.Error())
		}

		return fetchErr
	}); err != nil {
		log.Error("Failed, error: %s", err)
		os.Exit(1)
	}

	if git.ShouldCheckout() {
		if git.ShouldCheckoutTag() {
			if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
				if attempt > 0 {
					log.Warn("Retrying...")
				}

				fetchErr := git.FetchTags()
				if fetchErr != nil {
					log.Warn("%d attempt failed:", attempt)
					fmt.Println(fetchErr.Error())
				}

				return fetchErr
			}); err != nil {
				log.Error("Failed, error: %s", err)
				os.Exit(1)
			}
		}

		if err := git.Checkout(); err != nil {
			if !git.ShouldTryFetchUnshallow() {
				log.Error("Failed, error: %s", err)
				os.Exit(1)
			}

			log.Warn("Failed, error: %s", err)
			log.Warn("Unshallow...")

			if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
				if attempt > 0 {
					log.Warn("Retrying...")
				}

				fetchShallowErr := git.FetchUnshallow()
				if fetchShallowErr != nil {
					log.Warn("%d attempt failed:", attempt)
					fmt.Println(fetchShallowErr.Error())
				}

				return fetchShallowErr
			}); err != nil {
				log.Error("Failed, error: %s", err)
				os.Exit(1)
			}

			if err := git.Checkout(); err != nil {
				log.Error("Failed, error: %s", err)
				os.Exit(1)
			}
		}

		if git.ShouldMergePullRequest() {
			if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
				if attempt > 0 {
					log.Warn("Retrying...")
				}

				gitMergeErr := git.MergePullRequest()
				if gitMergeErr != nil {
					log.Warn("%d attempt failed:", attempt)
					fmt.Println(gitMergeErr.Error())
				}

				return gitMergeErr
			}); err != nil {
				log.Error("Failed, error: %s", err)
				os.Exit(1)
			}
		}

		if err := retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
			if attempt > 0 {
				log.Warn("Retrying...")
			}

			submoduleErr := git.SubmoduleUpdate()
			if submoduleErr != nil {
				log.Warn("%d attempt failed:", attempt)
				fmt.Println(submoduleErr.Error())
			}

			return submoduleErr
		}); err != nil {
			log.Error("Failed, error: %s", err)
			os.Exit(1)
		}

		log.Info("Exporting git logs")

		if commitHash, err := git.LogCommitHash(); err != nil {
			log.Error("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Detail("=> GIT_CLONE_COMMIT_HASH")
			log.Detail("   value: %s", commitHash)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_HASH", commitHash); err != nil {
				log.Warn("envman export failed, error: %s", err)
			}
		}

		if commitMessageSubject, err := git.LogCommitMessageSubject(); err != nil {
			log.Error("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Detail("=> GIT_CLONE_COMMIT_MESSAGE_SUBJECT")
			log.Detail("   value: %s", commitMessageSubject)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_MESSAGE_SUBJECT", commitMessageSubject); err != nil {
				log.Warn("envman export failed, error: %s", err)
			}
		}

		if commitMessageBody, err := git.LogCommitMessageBody(); err != nil {
			log.Error("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Detail("=> GIT_CLONE_COMMIT_MESSAGE_BODY")
			log.Detail("   value: %s", commitMessageBody)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_MESSAGE_BODY", commitMessageBody); err != nil {
				log.Warn("envman export failed, error: %s", err)
			}
		}

		if commitAuthorName, err := git.LogAuthorName(); err != nil {
			log.Error("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Detail("=> GIT_CLONE_COMMIT_AUTHOR_NAME")
			log.Detail("   value: %s", commitAuthorName)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_AUTHOR_NAME", commitAuthorName); err != nil {
				log.Warn("envman export failed, error: %s", err)
			}
		}

		if commitAuthorEmail, err := git.LogAuthorEmail(); err != nil {
			log.Error("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Detail("=> GIT_CLONE_COMMIT_AUTHOR_EMAIL")
			log.Detail("   value: %s", commitAuthorEmail)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_AUTHOR_EMAIL", commitAuthorEmail); err != nil {
				log.Warn("envman export failed, error: %s", err)
			}
		}

		if commitCommiterName, err := git.LogCommiterName(); err != nil {
			log.Error("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Detail("=> GIT_CLONE_COMMIT_COMMITER_NAME")
			log.Detail("   value: %s", commitCommiterName)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COMMITER_NAME", commitCommiterName); err != nil {
				log.Warn("envman export failed, error: %s", err)
			}
		}

		if commitCommiterEmail, err := git.LogCommiterEmail(); err != nil {
			log.Error("Git log failed, error: %s", err)
			os.Exit(1)
		} else {
			log.Detail("=> GIT_CLONE_COMMIT_COMMITER_EMAIL")
			log.Detail("   value: %s", commitCommiterEmail)
			fmt.Println()

			if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COMMITER_EMAIL", commitCommiterEmail); err != nil {
				log.Warn("envman export failed, error: %s", err)
			}
		}
	}

	log.Done("Success")
}
