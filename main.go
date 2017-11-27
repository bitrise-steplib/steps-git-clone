package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
)

const (
	retryCount = 2
	waitTime   = 5 // seconds
)

// vars ...
var (
	configs     ConfigsModel
	Git         *git.Git
	checkoutArg string
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

func init() {
	configs = createConfigsModelFromEnvs()
	fmt.Println()
	configs.print()
	if err := configs.validate(); err != nil {
		fail("Issue with input: %v", err)
	}
	fmt.Println()
	Git = git.New(configs.CloneIntoDir)
	checkoutArg = setCheckoutArg()

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

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := command.New("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

func run(c *command.Model) error {
	log.Infof(c.PrintableCommandArgs())
	return c.SetStdout(os.Stdout).SetStderr(os.Stderr).Run()
}

func runWithOutput(c *command.Model) (string, error) {
	//log.Infof(c.PrintableCommandArgs())
	return c.RunAndReturnTrimmedCombinedOutput()
}

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func isOriginPresent(dir, repoURL string) (bool, error) {
	absDir, err := pathutil.AbsPath(dir)
	if err != nil {
		return false, err
	}

	if file, err := os.Stat(absDir); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else if !file.IsDir() {
		return false, fmt.Errorf("file (%s) exists, but it's not a directory", dir)
	}

	remotes, err := runWithOutput(Git.RemoteList())
	if err != nil {
		return false, err
	}

	if !strings.Contains(remotes, repoURL) {
		return false, fmt.Errorf(".git folder exists in the directory (%s), but using a different remote", dir)
	}

	return true, nil
}

func resetRepo() error {
	if err := run(Git.Reset("--hard", "HEAD")); err != nil {
		return err
	}

	if err := run(Git.Clean("-x", "-d", "f")); err != nil {
		return err
	}

	if err := run(Git.SubmoduleForeach(Git.Reset("--hard", "HEAD"))); err != nil {
		return err
	}

	if err := run(Git.SubmoduleForeach(Git.Clean("-x", "-d", "-f"))); err != nil {
		return err
	}

	return nil
}

func isPR() bool {
	return configs.PullRequestURI != "" || configs.PullRequestID != "" || configs.PullRequestMergeBranch != ""
}

func setCheckoutArg() string {
	arg := ""
	if configs.Commit != "" {
		arg = configs.Commit
	} else if configs.Tag != "" {
		arg = configs.Tag
	} else if configs.Branch != "" {
		arg = configs.Branch
	}
	return arg
}

func getDiffFile() (string, error) {
	url := fmt.Sprintf("%s/diff.txt?api_token=%s", configs.BuildURL, configs.BuildAPIToken)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	} else if resp.StatusCode != 200 {
		return "", fmt.Errorf("Can't download diff file, HTTP status code: %d", resp.StatusCode)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("git-clone-step: failed to close response body, error: %s", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	diffFile, err := ioutil.TempFile("", configs.PullRequestID+".diff")
	if err != nil {
		return "", err
	}

	if _, err := diffFile.Write(body); err != nil {
		return "", err
	}
	if err := diffFile.Close(); err != nil {
		return "", err
	}

	return diffFile.Name(), nil
}

func runWithRetry(f func() *command.Model) error {
	return retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("Retrying...")
		}
		err := run(f())
		//err := run(cmd)
		if err != nil {
			log.Warnf("Attempt %d failed:", attempt+1)
			fmt.Println(err.Error())
		}

		return err
	})
}

func printLog(format, env string) error {
	l, err := runWithOutput(Git.Log(format))
	if err != nil {
		return err
	}

	log.Printf("=> %s\n   value: %s\n", env, l)
	if err := exportEnvironmentWithEnvman(env, l); err != nil {
		return fmt.Errorf("envman export failed, error: %v", err)
	}
	return nil
}

func main() {
	originPresent, err := isOriginPresent(configs.CloneIntoDir, configs.RepositoryURL)
	if err != nil {
		fail("git-clone-step: can't check if origin is presented, error: %v", err)
	}

	if originPresent && configs.ResetRepository == "yes" {
		if err := resetRepo(); err != nil {
			fail("git-clone-step: can't reset repository, error: %v", err)
		}
	}

	// Create directory if not exist
	if err := os.MkdirAll(configs.CloneIntoDir, 0755); err != nil {
		fail("git-clone-step: can't create directory (%s), error: %v", configs.CloneIntoDir, err)
	}

	if err := run(Git.Init()); err != nil {
		fail("git-clone-step: can't init repository, error: %v", err)
	}

	if !originPresent {
		if err := run(Git.RemoteAdd("origin", configs.RepositoryURL)); err != nil {
			fail("git-clone-step: can't add remote repository (%s), error: %v", configs.RepositoryURL, err)
		}
	}

	if err := runWithRetry(func() *command.Model {
		if configs.CloneDepth != "" {
			return Git.Fetch("--depth=" + configs.CloneDepth)
		}
		return Git.Fetch()
	}); err != nil {
		fail("git-clone-step: fetch failed, error: %v", err)
	}

	if isPR() {
		if configs.ManualMerge == "yes" {
			if err := run(Git.Checkout(configs.BranchDest)); err != nil {
				fail("git-clone-step: checkout failed (%s), error: %v", configs.BranchDest, err)
			}
			if err := run(Git.Merge(configs.Commit)); err != nil {
				log.Errorf("git-clone-step: merge failed (%s), error: %v", configs.Commit, err)
				if configs.PullRequestMergeBranch != "" {
					log.Warnf("Using Pull Request branch...")
					if err := runWithRetry(func() *command.Model {
						return Git.Fetch("origin", configs.PullRequestMergeBranch+":"+
							strings.TrimSuffix(configs.PullRequestMergeBranch, "/merge"))
					}); err != nil {
						fail("git-clone-step: fetch Pull Request branch failed (%s), error: %v",
							configs.PullRequestMergeBranch, err)
					}

					arg := strings.TrimSuffix(configs.PullRequestMergeBranch, "/merge")
					if err := run(Git.Checkout(arg)); err != nil {
						fail("git-clone-step: checkout failed (%s), error: %v", configs.BranchDest, err)
					}
				} else {
					log.Warnf("Applying patch...")
					patch, err := getDiffFile()
					if err != nil {
						fail("git-clone-step: can't download diff file, error: %v", err)
					}
					if err := run(Git.Apply(patch)); err != nil {
						fail("git-clone-step: can't apply patch (%s), error: %v", patch, err)
					}
				}
			}
		} else {
			if configs.PullRequestMergeBranch != "" {
				log.Warnf("Using Pull Request branch...")
				branch := strings.TrimSuffix(configs.PullRequestMergeBranch, "/merge")
				if err := run(Git.Fetch("origin", configs.PullRequestMergeBranch+":"+branch)); err != nil {
					fail("git-clone-step: fetch Pull Request branch failed (%s), error: %v",
						configs.PullRequestMergeBranch, err)
				}
				if err := run(Git.Checkout(branch)); err != nil {
					fail("git-clone-step: checkout failed (%s), error: %v", configs.BranchDest, err)
				}
			} else {
				if err := run(Git.Checkout(configs.BranchDest)); err != nil {
					fail("git-clone-step: checkout failed (%s), error: %v", configs.BranchDest, err)
				}
				log.Warnf("Applying patch...")
				patch, err := getDiffFile()
				if err != nil {
					fail("git-clone-step: can't download diff file, error: %v", err)
				}
				if err := run(Git.Apply(patch)); err != nil {
					fail("git-clone-step: can't apply patch (%s), error: %v", patch, err)
				}
			}
		}
	} else if checkoutArg != "" {
		if err := run(Git.Checkout(checkoutArg)); err != nil {
			if configs.CloneDepth == "" {
				fail("git-clone-step: checkout failed (%s), error: %v", checkoutArg, err)
			}
			log.Warnf("git-clone-step: checkout failed, error: %v\nUnshallow...", err)

			if err := runWithRetry(func() *command.Model {
				return Git.Fetch("--unshallow")
			}); err != nil {
				fail("git-clone-step: fetch failed, error: %v", err)
			}
			if err := run(Git.Checkout(checkoutArg)); err != nil {
				fail("git-clone-step: checkout failed (%s), error: %v", checkoutArg, err)
			}
		}
	}

	if configs.UpdateSubmodules == "yes" {
		if err := run(Git.SubmoduleUpdate()); err != nil {
			fail("git-clone-step: submodule update failed, error: %v", err)
		}
	}

	if checkoutArg != "" {
		log.Infof("\nExporting git logs\n")

		for format, env := range map[string]string{
			`"%H"`:  "GIT_CLONE_COMMIT_HASH",
			`"%s"`:  "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
			`"%b"`:  "GIT_CLONE_COMMIT_MESSAGE_BODY",
			`"%an"`: "GIT_CLONE_COMMIT_AUTHOR_NAME",
			`"%ae"`: "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
			`"%cn"`: "GIT_CLONE_COMMIT_COMMITER_NAME",
			`"%ce"`: "GIT_CLONE_COMMIT_COMMITER_EMAIL",
		} {
			if err := printLog(format, env); err != nil {
				fail("git-clone-step: git log failed, error: %v", err)
			}
		}

		count, err := runWithOutput(Git.RevList("HEAD", "--count"))
		if err != nil {
			fail("git-clone-step: git rev-list command failed, error: %v", err)
		}

		log.Printf("=> %s\n   value: %s\n", "GIT_CLONE_COMMIT_COUNT", count)
		if err := exportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COUNT", count); err != nil {
			fail("envman export failed, error: %v", err)
		}
	}

	log.Donef("Success")
}
