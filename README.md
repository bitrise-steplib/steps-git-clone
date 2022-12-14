# Git Clone Repository

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-git-clone?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-git-clone/releases)

The Step checks out the defined repository state, optionally updates the repository submodules and exports the achieved git repository state properties.

<details>
<summary>Description</summary>

The checkout process depends on the checkout properties: the Step either checks out a repository state defined by a git commit or a git tag, or achieves a merged state of a pull / merge request.
The Step uses two solutions to achieve the merged state of the pull / merge request: auto merge in the case of a merge branch or diff file (provided by the Git service) and manual merge otherwise.
Once the desired state is checked out, the Step optionally updates the submodules. In the case of pull / merge request, the Step checks out a detach head and exports the achieved git state properties.

### Configuring the Step

1. The **Git repository URL** and the ** Clone destination (local)directory path** fields are required fields and are automatically filled out based on your project settings.
Optionally, you can modify the following fields in the **Clone Config** section:
1. You can set the **Update the registered submodules?** option to `yes` to pull the most up-to-date version of the submodule from the submodule's repository.
2. You can set the number of commits you want the Step to fetch in the **Limit fetching to the specified number of commits** option. Make sure you set a decimal number.

Other **Clone config** inputs are not editable unless you go to the **bitrise.yml** tab, however, to avoid issues, we suggest you to contact our Support team instead.

### Troubleshooting
If you have GitHub Enterprise set up, it works slightly differently on [bitrise.io](https://www.bitrise.io) than on [github.com](https://github.com). You have to manually set the git clone URL, register the SSH key and the webhook.
If you face network issues in the case of self-hosted git servers, we advise you to contact our Support Team to help you out.
If you face slow clone speed, set the **Limit fetching to the specified number of commits** to the number of commits you want to clone instead of cloning the whole commit history or you can use the Git LFS solution provided by the git provider.

### Useful links

- [How to register a GitHub Enterprise repository](https://discuss.bitrise.io/t/how-to-register-a-github-enterprise-repository/218)
- [Code security](https://devcenter.bitrise.io/getting-started/code-security/)

### Related Steps

- [Activate SSH key (RSA private key)](https://www.bitrise.io/integrations/steps/activate-ssh-key)
- [Bitrise.io Cache:Pull](https://www.bitrise.io/integrations/steps/cache-pull)
- [Bitrise.io Cache:Push](https://www.bitrise.io/integrations/steps/cache-push)

</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `repository_url` | SSH or HTTPS URL of the repository to clone | required | `$GIT_REPOSITORY_URL` |
| `clone_into_dir` | Local directory where the repository is cloned | required | `$BITRISE_SOURCE_DIR` |
| `commit` | Commit SHA to checkout |  | `$BITRISE_GIT_COMMIT` |
| `tag` | Git tag to checkout |  | `$BITRISE_GIT_TAG` |
| `branch` | Git branch to checkout |  | `$BITRISE_GIT_BRANCH` |
| `branch_dest` | The branch that the pull request targets, such as `main` |  | `$BITRISEIO_GIT_BRANCH_DEST` |
| `pull_request_id` | Pull request number, coming from the Git provider |  | `$PULL_REQUEST_ID` |
| `pull_request_repository_url` | URL of the source repository of a pull request.  This points to the fork repository in builds triggered by pull requests. |  | `$BITRISEIO_PULL_REQUEST_REPOSITORY_URL` |
| `pull_request_merge_branch` | Git ref pointing to the result of merging the PR branch into the destination branch. Even if the source of the PR is a fork, this is a reference to the destination repository.  Example: `refs/pull/14/merge`  Note: not all Git services provide this value. |  | `$BITRISEIO_PULL_REQUEST_MERGE_BRANCH` |
| `pull_request_head_branch` | Git ref pointing to the head of the PR branch. Even if the source of the PR is a fork, this is a reference to the destination repository.  Example: `refs/pull/14/head`  Note: not all Git services provide this value. |  | `$BITRISEIO_PULL_REQUEST_HEAD_BRANCH` |
| `update_submodules` | Update the registered submodules to match what the superproject expects by cloning missing submodules, fetching missing commits in submodules and updating the working tree of the submodules. If set to "no" `git fetch` calls will get the `--no-recurse-submodules` flag. |  | `yes` |
| `clone_depth` | Limit fetching to the specified number of commits. The value should be a decimal number, for example `10`. |  |  |
| `submodule_update_depth` | Truncate the history to the specified number of revisions. The value should be a decimal number, for example `10`. |  |  |
| `merge_pr` | Disables merging the source and destination branches. - `yes`: The default setting. Merges the source branch into the destination branch. - `no`: Treats Pull Request events as Push events on the source branch. |  | `yes` |
| `sparse_directories` | Limit which directories should be cloned during the build. This could be useful if a repository contains multiple platforms, so called monorepositories, and the build is only targeting a single platform. For example, specifying "src/android" the Step will only clone: - contents of the root directory and - contents of the "src/android" directory and all subdirectories of "src/android". On the other hand, "src/ios" and any other directories will not be cloned. |  |  |
| `reset_repository` | Reset repository contents with `git reset --hard HEAD` and `git clean -f` before fetching. |  | `No` |
| `fetch_tags` | yes - fetch all tags from the remote by adding `--tags` flag to git fetch calls no - disable automatic tag following by adding `--no-tags` flag to git fetch calls |  | `no` |
| `build_url` | Unique build URL of this build on Bitrise.io |  | `$BITRISE_BUILD_URL` |
| `build_api_token` | The build's API Token for the build on Bitrise.io | sensitive | `$BITRISE_BUILD_API_TOKEN` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable               | Description                                                                                                                                                                                                                        |
|------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `GIT_CLONE_COMMIT_HASH`            | SHA hash of the checked-out commit.                                                                                                                                                                                                |
| `GIT_CLONE_COMMIT_MESSAGE_SUBJECT` | Commit message of the checked-out commit.                                                                                                                                                                                          |
| `GIT_CLONE_COMMIT_MESSAGE_BODY`    | Commit message body of the checked-out commit.                                                                                                                                                                                     |
| `GIT_CLONE_COMMIT_COUNT`           | Commit count after checkout.  Count will only work properly if no `--depth` option is set. If `--depth` is set then the history truncated to the specified number of commits. Count will **not** fail but will be the clone depth. |
| `GIT_CLONE_COMMIT_AUTHOR_NAME`     | Author of the checked-out commit.                                                                                                                                                                                                  |
| `GIT_CLONE_COMMIT_AUTHOR_EMAIL`    | Email of the checked-out commit.                                                                                                                                                                                                   |
| `GIT_CLONE_COMMIT_COMMITTER_NAME`  | Committer name of the checked-out commit.                                                                                                                                                                                          |
| `GIT_CLONE_COMMIT_COMMITTER_EMAIL` | Email of the checked-out commit.                                                                                                                                                                                                   |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-git-clone/pulls) and [issues](https://github.com/bitrise-steplib/steps-git-clone/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
