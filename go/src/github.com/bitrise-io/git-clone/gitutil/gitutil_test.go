package gitutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
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

func TestConfigureCheckoutParam(t *testing.T) {
	t.Log("it sets pullRequestID")
	{
		pullRequestID := "1"
		commitHash := ""
		tag := ""
		branch := ""
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "1", helper.pullRequestID)
		require.Equal(t, "pull/1", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
	}

	t.Log("it configures with commitHash")
	{
		pullRequestID := ""
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := ""
		branch := ""
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "", helper.pullRequestID)
		require.Equal(t, "670f2fe2ab44f8563c6784317a80bc07fad54634", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
	}

	t.Log("it configures with tag")
	{
		pullRequestID := ""
		commitHash := ""
		tag := "0.9.2"
		branch := ""
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "", helper.pullRequestID)
		require.Equal(t, "0.9.2", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
	}

	t.Log("it configures with branch")
	{
		pullRequestID := ""
		commitHash := ""
		tag := ""
		branch := "master"
		cloneDepth := ""

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "", helper.pullRequestID)
		require.Equal(t, "master", helper.checkoutParam)
		require.Equal(t, "", helper.cloneDepth)
	}

	t.Log("it configures with cloneDepth")
	{
		pullRequestID := ""
		commitHash := ""
		tag := ""
		branch := ""
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "", helper.pullRequestID)
		require.Equal(t, "", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > commitHash > tag > branch")
	{
		pullRequestID := "1"
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := "0.9.2"
		branch := "master"
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "1", helper.pullRequestID)
		require.Equal(t, "pull/1", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > commitHash > tag > branch")
	{
		pullRequestID := ""
		commitHash := "670f2fe2ab44f8563c6784317a80bc07fad54634"
		tag := "0.9.2"
		branch := "master"
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "", helper.pullRequestID)
		require.Equal(t, "670f2fe2ab44f8563c6784317a80bc07fad54634", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > commitHash > tag > branch")
	{
		pullRequestID := ""
		commitHash := ""
		tag := "0.9.2"
		branch := "master"
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "", helper.pullRequestID)
		require.Equal(t, "0.9.2", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}

	t.Log("it configures checkout with order of params - pullRequestID > commitHash > tag > branch")
	{
		pullRequestID := ""
		commitHash := ""
		tag := ""
		branch := "master"
		cloneDepth := "1"

		helper := Helper{}
		helper.ConfigureCheckoutParam(pullRequestID, commitHash, tag, branch, cloneDepth)
		require.Equal(t, "", helper.pullRequestID)
		require.Equal(t, "master", helper.checkoutParam)
		require.Equal(t, "1", helper.cloneDepth)
	}
}

func TestNewHelper(t *testing.T) {
	t.Log("it fails if destinationDir empty")
	{
		helper, err := NewHelper("", "https://github.com/bitrise-samples/git-clone-test.git")
		require.Error(t, err)
		require.Equal(t, "", helper.destinationDir)
		require.Equal(t, "", helper.remoteURI)
	}

	t.Log("it fails if remote URI empty")
	{
		helper, err := NewHelper("./", "")
		require.Error(t, err)
		require.Equal(t, "", helper.destinationDir)
		require.Equal(t, "", helper.remoteURI)
	}

	t.Log("it fails if remote URI empty")
	{
		helper, err := NewHelper("./", "")
		require.Error(t, err)
		require.Equal(t, "", helper.destinationDir)
		require.Equal(t, "", helper.remoteURI)
	}

	t.Log("it fails if destination dir contains .git dir")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__test__")
		require.NoError(t, err)

		destinationDir := filepath.Join(tmpDir, "dst")
		gitDir := filepath.Join(destinationDir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0777))

		helper, err := NewHelper(destinationDir, "https://github.com/bitrise-samples/git-clone-test.git")
		require.Error(t, err)
		require.Equal(t, "", helper.destinationDir)
		require.Equal(t, "", helper.remoteURI)

		require.NoError(t, os.RemoveAll(tmpDir))
	}

	t.Log("it creates destination dir if not exist")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__test__")
		require.NoError(t, err)

		destinationDir := filepath.Join(tmpDir, "dst")
		exist, err := pathutil.IsDirExists(destinationDir)
		require.NoError(t, err)
		require.Equal(t, false, exist)

		helper, err := NewHelper(destinationDir, "https://github.com/bitrise-samples/git-clone-test.git")
		require.NoError(t, err)
		require.Equal(t, destinationDir, helper.destinationDir)
		require.Equal(t, "https://github.com/bitrise-samples/git-clone-test.git", helper.remoteURI)

		exist, err = pathutil.IsDirExists(destinationDir)
		require.NoError(t, err)
		require.Equal(t, true, exist)

		require.NoError(t, os.RemoveAll(tmpDir))
	}
}
