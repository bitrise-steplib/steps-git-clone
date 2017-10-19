package gitutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
	"strings"
)

func TestProperReturn(t *testing.T) {
	t.Log("it returns nil without error")
	{
		require.NoError(t, properReturn(nil, ""))
	}

	t.Log("it returns nil without error")
	{
		require.NoError(t, properReturn(nil, "msg"))
	}

	t.Log("it returns error")
	{
		require.Error(t, properReturn(errors.New("error"), ""))
	}

	t.Log("it returns fallback message if error is exit status error and fallback message provided")
	{
		err := properReturn(errors.New("exit status 1"), "")
		require.Error(t, err)
		require.Equal(t, "exit status 1", err.Error())
	}

	t.Log("it returns fallback message if error is exit status error")
	{
		err := properReturn(errors.New("exit status 1"), "msg")
		require.Error(t, err)
		require.Equal(t, "msg", err.Error())
	}
}

func TestCreateGitLogCmdSlice(t *testing.T) {
	t.Log("it creates git log command")
	{
		cmdSlice := createGitLogCmdSlice(`--format="%H"`)
		require.Equal(t, 4, len(cmdSlice))
		require.Equal(t, "git", cmdSlice[0])
		require.Equal(t, "log", cmdSlice[1])
		require.Equal(t, "-1", cmdSlice[2])
		require.Equal(t, `--format="%H"`, cmdSlice[3])
	}
}

func TestCreateGitCmdSlice(t *testing.T) {
	t.Log("it creates git command")
	{
		cmdSlice := createGitCmdSlice("init")
		require.Equal(t, 2, len(cmdSlice))
		require.Equal(t, "git", cmdSlice[0])
		require.Equal(t, "init", cmdSlice[1])
	}
}

func TestConfigureCheckoutWithParams(t *testing.T) {
	t.Log("it sets pullRequestID")
	{
		pullRequestID := "1"
		pullRequestMergeBranch := "pull/1/merge"
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutWithPullRequestID(pullRequestID, pullRequestMergeBranch, cloneDepth, "")
		require.Equal(t, "1", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "pull/1", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
	}
	t.Log("it sets pullRequestID + patch arguments")
	{
		pullRequestID := "1"
		pullRequestMergeBranch := "pull/1/merge"
		cloneDepth := ""
		patchArgs := "--cached --index"

		helper := Helper{}
		helper.ConfigureCheckoutWithPullRequestID(pullRequestID, pullRequestMergeBranch, cloneDepth, patchArgs)
		require.Equal(t, "1", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "pull/1", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
		require.Equal(t, "--cached --index", strings.Join(helper.pullRequestHelper.PullRequestPatchArgs," "))
	}

	t.Log("it sets pullRequestRepositoryURI and pullRequestBranch")
	{
		pullRequestID := "1"
		pullRequestRepositoryURI := "https://github.com/bitrise-io/steps-git-clone.git"
		pullRequestBranch := "awesome-branch"
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutWithPullRequestURI(pullRequestID, pullRequestRepositoryURI, pullRequestBranch, cloneDepth, "")
		require.Equal(t, "https://github.com/bitrise-io/steps-git-clone.git", helper.pullRequestHelper.pullRequestRepositoryURI)
		require.Equal(t, "awesome-branch", helper.pullRequestHelper.pullRequestBranch)
		require.Equal(t, "", helper.cloneDepth)
		require.Equal(t, "",strings.Join(helper.pullRequestHelper.PullRequestPatchArgs, " "))
	}

	t.Log("it configures with commitHash")
	{
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := ""
		branch := ""
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutWithParams(commitHash, tag, branch, cloneDepth)
		require.Equal(t, "670f2fe2ab44f8563c6784317a80bc07fad54634", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
	}

	t.Log("it configures with tag")
	{
		commitHash := ""
		tag := "0.9.2"
		branch := ""
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutWithParams(commitHash, tag, branch, cloneDepth)
		require.Equal(t, "0.9.2", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
	}

	t.Log("it configures with branch")
	{
		commitHash := ""
		tag := ""
		branch := "master"
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutWithParams(commitHash, tag, branch, cloneDepth)
		require.Equal(t, "master", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
	}

	t.Log("it configures with cloneDepth")
	{
		commitHash := ""
		tag := ""
		branch := ""
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckoutWithParams(commitHash, tag, branch, cloneDepth)
		require.Equal(t, "", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > pullRequest > commitHash > tag > branch")
	{
		pullRequestID := "1"
		pullRequestRepositoryURI := "https://github.com/bitrise-io/steps-git-clone.git"
		pullRequestMergeBranch := "pull/1/merge"
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := "0.9.2"
		branch := "master"
		branchDest := "feature/awesome-branch"
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckout(pullRequestID, pullRequestRepositoryURI, pullRequestMergeBranch, commitHash, tag, branch, branchDest, cloneDepth, "", "", "")
		require.Equal(t, "1", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestRepositoryURI)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestBranch)
		require.Equal(t, "pull/1", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > pullRequest > commitHash > tag > branch")
	{
		pullRequestID := "1"
		pullRequestRepositoryURI := "https://github.com/bitrise-io/steps-git-clone.git"
		pullRequestMergeBranch := ""
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := "0.9.2"
		branch := "feature/awesome-branch"
		branchDest := "master"
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckout(pullRequestID, pullRequestRepositoryURI, pullRequestMergeBranch, commitHash, tag, branch, branchDest, cloneDepth, "", "", "")
		require.Equal(t, "1", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "https://github.com/bitrise-io/steps-git-clone.git", helper.remoteURI)
		require.Equal(t, "master", helper.pullRequestHelper.pullRequestBranch)
		require.Equal(t, "670f2fe2ab44f8563c6784317a80bc07fad54634", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > pullRequest > commitHash > tag > branch + patch arguments")
	{
		pullRequestID := "1"
		pullRequestRepositoryURI := "https://github.com/bitrise-io/steps-git-clone.git"
		pullRequestMergeBranch := "pull/1/merge"
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := "0.9.2"
		branch := "master"
		branchDest := "feature/awesome-branch"
		cloneDepth := "1"
		patchArgs:= "--cached"

		helper := Helper{}
		helper.ConfigureCheckout(pullRequestID, pullRequestRepositoryURI, pullRequestMergeBranch, commitHash, tag, branch, branchDest, cloneDepth, "", "", patchArgs)
		require.Equal(t, "1", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestRepositoryURI)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestBranch)
		require.Equal(t, "pull/1", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
		require.Equal(t, "--cached", strings.Join(helper.pullRequestHelper.PullRequestPatchArgs," "))
	}

	t.Log("it configures checkout with order of params - pullRequestID > pullRequest > commitHash > tag > branch + patch arguments")
	{
		pullRequestID := "1"
		pullRequestRepositoryURI := "https://github.com/bitrise-io/steps-git-clone.git"
		pullRequestMergeBranch := ""
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := "0.9.2"
		branch := "feature/awesome-branch"
		branchDest := "master"
		cloneDepth := "1"
		patchArgs := "--cached"

		helper := Helper{}
		helper.ConfigureCheckout(pullRequestID, pullRequestRepositoryURI, pullRequestMergeBranch, commitHash, tag, branch, branchDest, cloneDepth, "", "", patchArgs)
		require.Equal(t, "1", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "https://github.com/bitrise-io/steps-git-clone.git", helper.remoteURI)
		require.Equal(t, "master", helper.pullRequestHelper.pullRequestBranch)
		require.Equal(t, "670f2fe2ab44f8563c6784317a80bc07fad54634", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
		require.Equal(t, []string{"--cached"}, helper.pullRequestHelper.PullRequestPatchArgs)
	}

	t.Log("it configures checkout with order of params - pullRequestID > commitHash > tag > branch")
	{
		pullRequestID := ""
		pullRequestRepositoryURI := ""
		pullRequestMergeBranch := ""
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := "0.9.2"
		branch := "master"
		branchDest := ""
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckout(pullRequestID, pullRequestRepositoryURI, pullRequestMergeBranch, commitHash, tag, branch, branchDest, cloneDepth, "", "", "")
		require.Equal(t, "", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestRepositoryURI)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestBranch)
		require.Equal(t, "670f2fe2ab44f8563c6784317a80bc07fad54634", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > commitHash > tag > branch")
	{
		pullRequestID := ""
		pullRequestRepositoryURI := ""
		pullRequestMergeBranch := ""
		commitHash := ""
		tag := "0.9.2"
		branch := "master"
		branchDest := ""
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckout(pullRequestID, pullRequestRepositoryURI, pullRequestMergeBranch, commitHash, tag, branch, branchDest, cloneDepth, "", "", "")
		require.Equal(t, "", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestRepositoryURI)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestBranch)
		require.Equal(t, "0.9.2", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > commitHash > tag > branch")
	{
		pullRequestID := ""
		pullRequestRepositoryURI := ""
		pullRequestMergeBranch := ""
		commitHash := ""
		tag := ""
		branch := "master"
		branchDest := ""
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckout(pullRequestID, pullRequestRepositoryURI, pullRequestMergeBranch, commitHash, tag, branch, branchDest, cloneDepth, "", "", "")
		require.Equal(t, "", helper.pullRequestHelper.pullRequestID)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestRepositoryURI)
		require.Equal(t, "", helper.pullRequestHelper.pullRequestBranch)
		require.Equal(t, "master", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}
}

func TestNewHelper(t *testing.T) {
	t.Log("it fails if destinationDir empty")
	{
		helper, err := NewHelper("", "https://github.com/bitrise-samples/git-clone-test.git", false)
		require.Error(t, err)
		require.Equal(t, "", helper.destinationDir)
		require.Equal(t, "", helper.remoteURI)
	}

	t.Log("it fails if remote URI empty")
	{
		helper, err := NewHelper("./", "", false)
		require.Error(t, err)
		require.Equal(t, "", helper.destinationDir)
		require.Equal(t, "", helper.remoteURI)
	}

	t.Log("it fails if remote URI empty")
	{
		helper, err := NewHelper("./", "", false)
		require.Error(t, err)
		require.Equal(t, "", helper.destinationDir)
		require.Equal(t, "", helper.remoteURI)
	}

	t.Log("it creates destination dir if not exist")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__test__")
		require.NoError(t, err)

		destinationDir := filepath.Join(tmpDir, "dst")
		exist, err := pathutil.IsDirExists(destinationDir)
		require.NoError(t, err)
		require.Equal(t, false, exist)

		helper, err := NewHelper(destinationDir, "https://github.com/bitrise-samples/git-clone-test.git", false)
		require.NoError(t, err)
		require.Equal(t, destinationDir, helper.destinationDir)
		require.Equal(t, "https://github.com/bitrise-samples/git-clone-test.git", helper.remoteURI)

		exist, err = pathutil.IsDirExists(destinationDir)
		require.NoError(t, err)
		require.Equal(t, true, exist)

		require.NoError(t, os.RemoveAll(tmpDir))
	}
}
