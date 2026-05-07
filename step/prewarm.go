package step

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cachefile "github.com/bitrise-io/bitrise-build-cache-cli/v2/pkg/file"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

// PrewarmRepoFromBuildCache attempts to populate cloneIntoDir with a pre-built
// .git directory restored from the Bitrise Build Cache, mirroring the
// `cached-tar` HIT branch from the hackathon-2026-git-prewarm workflow.
//
// On any error or cache miss it logs the cause and returns nil — callers
// should fall back to a normal clone.
func PrewarmRepoFromBuildCache(ctx context.Context, logger log.Logger, envRepo env.Repository, cloneIntoDir, repositoryURL string) error {
	overallStart := time.Now()

	logger.Println()
	logger.Infof("Git repo prewarm: trying to restore tarball from Bitrise Build Cache")

	appSlug := envRepo.Get("BITRISE_APP_SLUG")
	if appSlug == "" {
		logger.Infof("BITRISE_APP_SLUG is not set — skipping prewarm")
		return nil
	}

	cacheKey := fmt.Sprintf("gitrepo-%s-main", appSlug)
	tarFile := filepath.Join(os.TempDir(), cacheKey+".tar")
	logger.Infof("Cache key: %s", cacheKey)
	logger.Infof("Tar file: %s", tarFile)
	logger.Infof("Clone dir: %s", cloneIntoDir)

	// Always try to clean up the downloaded tar file when we're done with it.
	defer func() {
		if err := os.Remove(tarFile); err != nil && !os.IsNotExist(err) {
			logger.Warnf("Failed to remove tar file %s: %s", tarFile, err)
		}
	}()

	// Make sure no stale tar is lying around from a previous run.
	if err := os.Remove(tarFile); err != nil && !os.IsNotExist(err) {
		logger.Warnf("Failed to remove pre-existing tar file %s: %s", tarFile, err)
	}

	helper := cachefile.NewHelper(cachefile.HelperParams{
		Logger: logger,
	})

	downloadStart := time.Now()
	logger.Infof("Downloading tarball from build cache...")
	if err := helper.Restore(ctx, cacheKey, tarFile); err != nil {
		if errors.Is(err, cachefile.ErrCacheNotFound) {
			logger.Infof("Build cache MISS for key %q — falling back to normal clone", cacheKey)
			return nil
		}
		logger.Warnf("Failed to restore tarball from build cache: %s — falling back to normal clone", err)
		return nil
	}
	downloadDuration := time.Since(downloadStart).Round(time.Millisecond)

	tarInfo, err := os.Stat(tarFile)
	if err != nil {
		logger.Warnf("Failed to stat downloaded tar file: %s — falling back to normal clone", err)
		return nil
	}
	if tarInfo.Size() == 0 {
		logger.Warnf("Downloaded tarball is empty — falling back to normal clone")
		return nil
	}
	logger.Infof("Build cache HIT: %d bytes in %s", tarInfo.Size(), downloadDuration)

	if err := os.MkdirAll(cloneIntoDir, 0755); err != nil {
		logger.Warnf("Failed to create clone dir %s: %s — falling back to normal clone", cloneIntoDir, err)
		return nil
	}

	extractStart := time.Now()
	logger.Infof("Extracting tarball into %s", cloneIntoDir)
	if out, err := runCmd("", "tar", "-xf", tarFile, "-C", cloneIntoDir); err != nil {
		logger.Warnf("Failed to extract tarball: %s\n%s\nFalling back to normal clone", err, out)
		return nil
	}
	logger.Infof("Tarball extracted in %s", time.Since(extractStart).Round(time.Millisecond))

	branchStart := time.Now()
	logger.Infof("Reading default branch via git symbolic-ref --short HEAD")
	branchOut, err := runCmd(cloneIntoDir, "git", "symbolic-ref", "--short", "HEAD")
	if err != nil {
		logger.Warnf("Failed to read default branch: %s\n%s\nFalling back to normal clone", err, branchOut)
		return nil
	}
	branch := strings.TrimSpace(branchOut)
	if branch == "" {
		logger.Warnf("Default branch resolved to empty string — falling back to normal clone")
		return nil
	}
	logger.Infof("Default branch: %s (resolved in %s)", branch, time.Since(branchStart).Round(time.Millisecond))

	logger.Infof("Adding origin remote -> %s", repositoryURL)
	if out, err := runCmd(cloneIntoDir, "git", "remote", "add", "origin", repositoryURL); err != nil {
		logger.Warnf("Failed to add origin remote: %s\n%s\nFalling back to normal clone", err, out)
		return nil
	}

	checkoutStart := time.Now()
	logger.Infof("Initializing working tree via git checkout %s -- .", branch)
	if out, err := runCmd(cloneIntoDir, "git", "checkout", branch, "--", "."); err != nil {
		logger.Warnf("Failed to initialize working tree: %s\n%s\nFalling back to normal clone", err, out)
		return nil
	}
	logger.Infof("Working tree initialized in %s", time.Since(checkoutStart).Round(time.Millisecond))

	logger.Donef("Git repo prewarm complete in %s", time.Since(overallStart).Round(time.Millisecond))
	return nil
}

func runCmd(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
