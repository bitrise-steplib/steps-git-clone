package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

func isPR() bool {
	return config.PRRepositoryCloneURL != "" || config.PRID != "" || config.PRMergeBranch != ""
}

func getCheckoutArg() string {
	arg := ""
	if config.Commit != "" {
		arg = config.Commit
	} else if config.Tag != "" {
		arg = config.Tag
	} else if config.Branch != "" {
		arg = config.Branch
	}
	return arg
}

func getDiffFile() (string, error) {
	url := fmt.Sprintf("%s/diff.txt?api_token=%s", config.BuildURL, config.BuildAPIToken)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	} else if resp.StatusCode != 200 {
		return "", fmt.Errorf("Can't download diff file, HTTP status code: %d", resp.StatusCode)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body, error: %s", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	diffFile, err := ioutil.TempFile("", fmt.Sprintf("%s.diff", config.PRID))
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
	// log.Infof(c.PrintableCommandArgs())
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

func isFork() bool {
	return config.PRRepositoryCloneURL != "" &&
		config.RepositoryURL != config.PRRepositoryCloneURL
}

func isPrivate() bool {
	return strings.HasPrefix(config.PRRepositoryCloneURL, "git")
}

func autoMerge() error {
	if err := runWithRetry(func() *command.Model {
		if config.CloneDepth != "" {
			return Git.Fetch("--depth=" + config.CloneDepth)
		}
		return Git.Fetch()
	}); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if config.PRMergeBranch != "" {
		if err := runWithRetry(func() *command.Model {
			return Git.Fetch("origin", config.PRMergeBranch+":"+
				strings.TrimSuffix(config.PRMergeBranch, "/merge"))
		}); err != nil {
			return fmt.Errorf("fetch Pull Request branch failed (%s), error: %v",
				config.PRMergeBranch, err)
		}

		arg := strings.TrimSuffix(config.PRMergeBranch, "/merge")
		if err := run(Git.Checkout(arg)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", config.BranchDest, err)
		}
	} else if patch, err := getDiffFile(); err == nil {
		if err := run(Git.Checkout(config.BranchDest)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", config.BranchDest, err)
		}
		if err := run(Git.Apply(patch)); err != nil {
			return fmt.Errorf("can't apply patch (%s), error: %v", patch, err)
		}
	} else {
		return fmt.Errorf("there is no Pull Request branch and can't download diff file")
	}
	return nil
}

func manualMerge() error {
	if err := runWithRetry(func() *command.Model {
		if config.CloneDepth != "" {
			return Git.Fetch("--depth=" + config.CloneDepth)
		}
		return Git.Fetch()
	}); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if err := run(Git.Checkout(config.BranchDest)); err != nil {
		return fmt.Errorf("checkout failed (%s), error: %v", config.BranchDest, err)
	}

	if isFork() {
		if err := run(Git.RemoteAdd("upstream", config.PRRepositoryCloneURL)); err != nil {
			return fmt.Errorf("couldn't add remote (%s), error: %v", config.PRRepositoryCloneURL, err)
		}

		if err := runWithRetry(func() *command.Model {
			return Git.Fetch("upstream", config.Branch)
		}); err != nil {
			return fmt.Errorf("fetch Pull Request branch failed (%s), error: %v",
				config.Branch, err)
		}

		if err := run(Git.Merge("upstream/" + config.Branch)); err != nil {
			return fmt.Errorf("merge failed (upstream/%s), error: %v", config.Branch, err)
		}
	} else {
		if err := run(Git.Merge(config.Commit)); err != nil {
			return fmt.Errorf("merge failed (%s), error: %v", config.Commit, err)
		}
	}

	return nil
}

func checkout(arg string) error {
	if err := runWithRetry(func() *command.Model {
		if config.CloneDepth != "" {
			return Git.Fetch("--depth=" + config.CloneDepth)
		}
		return Git.Fetch()
	}); err != nil {
		return fmt.Errorf("Fetch failed, error: %v", err)
	}

	if err := run(Git.Checkout(arg)); err != nil {
		if config.CloneDepth == "" {
			return fmt.Errorf("checkout failed (%s), error: %v", checkoutArg, err)
		}
		log.Warnf("Checkout failed, error: %v\nUnshallow...", err)

		if err := runWithRetry(func() *command.Model {
			return Git.Fetch("--unshallow")
		}); err != nil {
			return fmt.Errorf("fetch failed, error: %v", err)
		}
		if err := run(Git.Checkout(checkoutArg)); err != nil {
			return fmt.Errorf("checkout failed (%s), error: %v", checkoutArg, err)
		}
	}

	return nil
}
