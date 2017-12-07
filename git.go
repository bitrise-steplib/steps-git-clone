package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
)

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

	remotes, err := runForOutput(Git.RemoteList())
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

	if err := run(Git.Clean("-x", "-d", "-f")); err != nil {
		return err
	}

	if err := run(Git.SubmoduleForeach(Git.Reset("--hard", "HEAD"))); err != nil {
		return err
	}

	return run(Git.SubmoduleForeach(Git.Clean("-x", "-d", "-f")))
}

func isPR(prRepoURL, prMergeBranch string, prID int) bool {
	return prRepoURL != "" || prID != 0 || prMergeBranch != ""
}

func getCheckoutArg(commit, tag, branch string) string {
	arg := ""
	if commit != "" {
		arg = commit
	} else if tag != "" {
		arg = tag
	} else if branch != "" {
		arg = branch
	}
	return arg
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

func runForOutput(c *command.Model) (string, error) {
	return c.RunAndReturnTrimmedCombinedOutput()
}

func runWithRetry(f func() *command.Model) error {
	return retry.Times(retryCount).Wait(waitTime).Try(func(attempt uint) error {
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
	return prRepoURL != "" && repoURL != prRepoURL
}

func isPrivate(repoURL string) bool {
	return strings.HasPrefix(repoURL, "git")
}

func autoMerge(mergeBranch, branchDest, buildURL, apiToken string, depth, id int) error {
	if err := runWithRetry(func() *command.Model {
		if depth != 0 {
			return Git.Fetch("--depth=" + strconv.Itoa(depth))
		}
		return Git.Fetch()
	}); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if mergeBranch != "" {
		if err := runWithRetry(func() *command.Model {
			return Git.Fetch("origin", mergeBranch+":"+
				strings.TrimSuffix(mergeBranch, "/merge"))
		}); err != nil {
			return fmt.Errorf("fetch Pull Request branch failed (%s), error: %v",
				mergeBranch, err)
		}

		arg := strings.TrimSuffix(mergeBranch, "/merge")
		if err := run(Git.Checkout(arg)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", branchDest, err)
		}
	} else if patch, err := getDiffFile(buildURL, apiToken, id); err == nil {
		if err := run(Git.Checkout(branchDest)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", branchDest, err)
		}
		if err := run(Git.Apply(patch)); err != nil {
			return fmt.Errorf("can't apply patch (%s), error: %v", patch, err)
		}
	} else {
		return fmt.Errorf("there is no Pull Request branch and can't download diff file")
	}
	return nil
}

func manualMerge(repoURL, prRepoURL, branch, commit, branchDest string, depth int) error {
	if err := runWithRetry(func() *command.Model {
		if depth != 0 {
			return Git.Fetch("--depth=" + strconv.Itoa(depth))
		}
		return Git.Fetch()
	}); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if err := run(Git.Checkout(branchDest)); err != nil {
		return fmt.Errorf("checkout failed (%s), error: %v", branchDest, err)
	}

	if isFork(repoURL, prRepoURL) {
		if err := run(Git.RemoteAdd("fork", prRepoURL)); err != nil {
			return fmt.Errorf("couldn't add remote (%s), error: %v", prRepoURL, err)
		}

		if err := runWithRetry(func() *command.Model {
			return Git.Fetch("fork", branch)
		}); err != nil {
			return fmt.Errorf("fetch Pull Request branch failed (%s), error: %v",
				branch, err)
		}

		if err := run(Git.Merge("fork/" + branch)); err != nil {
			return fmt.Errorf("merge failed (fork/%s), error: %v", branch, err)
		}
	} else {
		if err := run(Git.Merge(commit)); err != nil {
			return fmt.Errorf("merge failed (%s), error: %v", commit, err)
		}
	}

	return nil
}

func checkout(arg string, depth int) error {
	if err := runWithRetry(func() *command.Model {
		if depth != 0 {
			return Git.Fetch("--depth=" + strconv.Itoa(depth))
		}
		return Git.Fetch()
	}); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if err := run(Git.Checkout(arg)); err != nil {
		if depth == 0 {
			return fmt.Errorf("checkout failed (%s), error: %v", arg, err)
		}
		log.Warnf("Checkout failed, error: %v\nUnshallow...", err)

		if err := runWithRetry(func() *command.Model {
			return Git.Fetch("--unshallow")
		}); err != nil {
			return fmt.Errorf("fetch failed, error: %v", err)
		}
		if err := run(Git.Checkout(arg)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", arg, err)
		}
	}

	return nil
}
