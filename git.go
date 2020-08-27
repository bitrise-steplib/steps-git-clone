package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
)

func isOriginPresent(gitCmd git.Git, dir, repoURL string) (bool, error) {
	absDir, err := pathutil.AbsPath(dir)
	if err != nil {
		return false, err
	}

	gitDir := filepath.Join(absDir, ".git")
	if exist, err := pathutil.IsDirExists(gitDir); err != nil {
		return false, err
	} else if exist {
		remotes, err := output(gitCmd.RemoteList())
		if err != nil {
			return false, err
		}

		if !strings.Contains(remotes, repoURL) {
			return false, fmt.Errorf(".git folder exists in the directory (%s), but using a different remote", dir)
		}
		return true, nil
	}

	return false, nil
}

func resetRepo(gitCmd git.Git) error {
	if err := run(gitCmd.Reset("--hard", "HEAD")); err != nil {
		return err
	}
	if err := run(gitCmd.Clean("-x", "-d", "-f")); err != nil {
		return err
	}
	if err := run(gitCmd.SubmoduleForeach(gitCmd.Reset("--hard", "HEAD"))); err != nil {
		return err
	}
	return run(gitCmd.SubmoduleForeach(gitCmd.Clean("-x", "-d", "-f")))
}

func getCheckoutArg(commit, tag, branch string) string {
	switch {
	case commit != "":
		return commit
	case tag != "":
		return tag
	case branch != "":
		return branch
	default:
		return ""
	}
}

func getDiffFile(buildURL, apiToken string, prID int) (string, error) {
	url := fmt.Sprintf("%s/diff.txt?api_token=%s", buildURL, apiToken)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body, error: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Can't download diff file, HTTP status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	diffFile, err := ioutil.TempFile("", fmt.Sprintf("%d.diff", prID))
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

func run(c *command.Model) error {
	log.Infof(c.PrintableCommandArgs())
	return c.SetStdout(os.Stdout).SetStderr(os.Stderr).Run()
}

func output(c *command.Model) (string, error) {
	return c.RunAndReturnTrimmedCombinedOutput()
}

func runWithRetry(f func() *command.Model) error {
	return retry.Times(2).Wait(5).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("Retrying...")
		}

		err := run(f())
		if err != nil {
			log.Warnf("Attempt %d failed:", attempt+1)
			fmt.Println(err.Error())
		}

		return err
	})
}

func isFork(repoURL, prRepoURL string) bool {
	return prRepoURL != "" && getRepo(repoURL) != getRepo(prRepoURL)
}

// formats:
// https://hostname/owner/repository.git
// git@hostname:owner/repository.git
// ssh://git@hostname:port/owner/repository.git
func getRepo(url string) string {
	var host, repo string
	switch {
	case strings.HasPrefix(url, "https://"):
		url = strings.TrimPrefix(url, "https://")
		idx := strings.Index(url, "/")
		host, repo = url[:idx], url[idx+1:]
	case strings.HasPrefix(url, "git@"):
		url = url[strings.Index(url, "@")+1:]
		idx := strings.Index(url, ":")
		host, repo = url[:idx], url[idx+1:]
	case strings.HasPrefix(url, "ssh://"):
		url = url[strings.Index(url, "@")+1:]
		if strings.Contains(url, ":") {
			idxColon, idxSlash := strings.Index(url, ":"), strings.Index(url, "/")
			host, repo = url[:idxColon], url[idxSlash+1:]
		} else {
			idx := strings.Index(url, "/")
			host, repo = url[:idx], url[idx+1:]
		}
	}
	return host + "/" + strings.TrimSuffix(repo, ".git")
}

func isPrivate(repoURL string) bool {
	return strings.HasPrefix(repoURL, "git")
}

// fetchArg converts the incoming mergeBranch pull/x/merge to pull/x/head:pull/x
// where x is the pull request id.
func fetchArg(mergeBranch string) string {
	arg := strings.TrimSuffix(mergeBranch, "/merge")
	return strings.Replace(mergeBranch, "merge", "head", 1) + ":" + arg
}

func mergeArg(mergeBranch string) string {
	return strings.TrimSuffix(mergeBranch, "/merge")
}

func autoMerge(gitCmd git.Git, mergeBranch, branchDest, buildURL, apiToken string, depth, id int) error {
	if err := runWithRetry(func() *command.Model {
		var opts []string
		if depth != 0 {
			opts = append(opts, "--depth="+strconv.Itoa(depth))
		}
		opts = append(opts, "origin", branchDest)
		return gitCmd.Fetch(opts...)
	}); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if mergeBranch != "" {
		if err := runWithRetry(func() *command.Model {
			return gitCmd.Fetch("origin", fetchArg(mergeBranch))
		}); err != nil {
			return fmt.Errorf("fetch Pull Request branch failed (%s), error: %v",
				mergeBranch, err)
		}
		if err := pull(gitCmd, branchDest); err != nil {
			return fmt.Errorf("pull failed (%s), error: %v", branchDest, err)
		}
		log.Printf("AAA gitCmd.Merge")
		if err := run(gitCmd.Merge(mergeArg(mergeBranch))); err != nil {
			if depth == 0 {
				return fmt.Errorf("merge %q: %v", mergeArg(mergeBranch), err)
			}
			log.Warnf("Merge failed, error: %v\nReset repository, then unshallow...", err)
			if err := resetRepo(gitCmd); err != nil {
				return fmt.Errorf("reset repository, error: %v", err)
			}
			if err := runWithRetry(func() *command.Model {
				return gitCmd.Fetch("--unshallow")
			}); err != nil {
				return fmt.Errorf("fetch failed, error: %v", err)
			}
			log.Printf("BBB gitCmd.Merge")
			if err := run(gitCmd.Merge(mergeArg(mergeBranch))); err != nil {
				return fmt.Errorf("merge %q: %v", mergeArg(mergeBranch), err)
			}
		}
	} else if patch, err := getDiffFile(buildURL, apiToken, id); err == nil {
		if err := run(gitCmd.Checkout(branchDest)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", branchDest, err)
		}
		if err := run(gitCmd.Apply(patch)); err != nil {
			return fmt.Errorf("can't apply patch (%s), error: %v", patch, err)
		}
	} else {
		return fmt.Errorf("there is no Pull Request branch and can't download diff file")
	}
	return nil
}

func manualMerge(gitCmd git.Git, repoURL, prRepoURL, branch, commit, branchDest string) error {
	if err := runWithRetry(func() *command.Model { return gitCmd.Fetch("origin", branchDest) }); err != nil {
		return fmt.Errorf("fetch failed, error: %v", err)
	}
	if err := pull(gitCmd, branchDest); err != nil {
		return fmt.Errorf("pull failed (%s), error: %v", branchDest, err)
	}
	commitHash, err := output(gitCmd.Log("%H"))
	if err != nil {
		log.Errorf("log commit hash: %v", err)
	}
	log.Printf("commit hash: %s", commitHash)

	if isFork(repoURL, prRepoURL) {
		if err := run(gitCmd.RemoteAdd("fork", prRepoURL)); err != nil {
			return fmt.Errorf("couldn't add remote (%s), error: %v", prRepoURL, err)
		}
		if err := runWithRetry(func() *command.Model { return gitCmd.Fetch("fork", branch) }); err != nil {
			return fmt.Errorf("fetch Pull Request branch failed (%s), error: %v", branch, err)
		}
		if err := run(gitCmd.Merge("fork/" + branch)); err != nil {
			return fmt.Errorf("merge failed (fork/%s), error: %v", branch, err)
		}
	} else {
		if err := run(gitCmd.Fetch("origin", branch)); err != nil {
			return fmt.Errorf("fetch failed, error: %v", err)
		}
		
		log.Printf("CCC gitCmd.Merge")
		if err := run(gitCmd.Merge(commit)); err != nil {
			return fmt.Errorf("merge failed (%s), error: %v", commit, err)
		}
	}

	return nil
}

func checkout(gitCmd git.Git, arg, branch string, depth int, isTag bool) error {
	if err := runWithRetry(func() *command.Model {
		var opts []string
		if depth != 0 {
			opts = append(opts, "--depth="+strconv.Itoa(depth))
		}
		if isTag {
			opts = append(opts, "--tags")
		}
		if branch != "" {
			opts = append(opts, "origin", branch)
		}
		return gitCmd.Fetch(opts...)
	}); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if err := run(gitCmd.Checkout(arg)); err != nil {
		if depth == 0 {
			return fmt.Errorf("checkout failed (%s), error: %v", arg, err)
		}
		log.Warnf("Checkout failed, error: %v\nUnshallow...", err)
		if err := runWithRetry(func() *command.Model {
			return gitCmd.Fetch("--unshallow")
		}); err != nil {
			return fmt.Errorf("fetch failed, error: %v", err)
		}
		if err := run(gitCmd.Checkout(arg)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", arg, err)
		}
	}

	return nil
}

// pull is a 'git fetch' followed by a 'git merge' which is the same as 'git pull'.
func pull(gitCmd git.Git, branchDest string) error {
	if err := run(gitCmd.Checkout(branchDest)); err != nil {
		return fmt.Errorf("checkout failed (%s), error: %v", branchDest, err)
	}
	log.Printf("DDD gitCmd.Merge")
	return run(gitCmd.Merge("origin/" + branchDest))
}
