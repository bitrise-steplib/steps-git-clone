# Git Clone Repository

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-git-clone?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-git-clone/releases)

Checks out the repository, updates submodules and exports git metadata as Step outputs.

<details>
<summary>Description</summary>

The checkout process depends on the Step settings and the build trigger parameters (coming from your git server).

Depending on the conditions, the step can checkout:
- the merged state of a Pull Request
- the head of a Pull Request
- a git tag
- a specific commit on a branch
- the head of a branch

The Step also supports more advanced features, such as updating submodules and sparse checkouts.

### Configuring the Step

The step should work with its default configuration if build triggers and webhooks are set up correctly.

By default, the Step performs a shallow clone in most cases (fetching only the latest commit) to make the clone fast and efficient. If your workflow requires a deeper commit history, you can override this using the **Clone depth** input.

### Useful links

- [How to register a GitHub Enterprise repository](https://discuss.bitrise.io/t/how-to-register-a-github-enterprise-repository/218)
- [Code security](https://devcenter.bitrise.io/getting-started/code-security/)

### Related Steps

- [Activate SSH key (RSA private key)](https://www.bitrise.io/integrations/steps/activate-ssh-key)
- [Generate changelog](https://bitrise.io/integrations/steps/generate-changelog)

</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `merge_pr` | This only applies to builds triggered by pull requests.  Options: - `yes`: Depending on the information in the build trigger, either fetches the PR merge ref or creates the merged state locally. - `no`: Checks out the head of the PR branch without merging it into the destination branch. |  | `yes` |
| `clone_into_dir` | Local directory where the repository is cloned | required | `$BITRISE_SOURCE_DIR` |
| `clone_depth` | Limit fetching to the specified number of commits.  By default, the Step tries to do a shallow clone (depth of 1) if it's possible based on the build trigger parameters. If it's not possible, it applies a low depth value, unless another value is specified here.  It's not recommended to define this input because a shallow clone ensures fast clone times. Examples of when you want to override the clone depth:  - A Step in the workflow reads the commit history in order to generate a changelog - A Step in the workflow runs a git diff against a previous commit |  |  |
| `update_submodules` | Update registered submodules to match what the superproject expects. If set to `no`, `git fetch` calls will use the `--no-recurse-submodules` flag. |  | `yes` |
| `submodule_update_depth` | When updating submodules, limit fetching to the specified number of commits. The value should be a decimal number, for example `10`. |  |  |
| `fetch_tags` | yes - fetch all tags from the remote by adding `--tags` flag to `git fetch` calls no - disable automatic tag following by adding `--no-tags` flag to `git fetch` calls |  | `no` |
| `sparse_directories` | Limit which directories to clone using [sparse-checkout](https://git-scm.com/docs/git-sparse-checkout). This is useful for monorepos where the current workflow only needs a subfolder.  For example, specifying `src/android` the Step will only clone: - contents of the root directory and - contents of the `src/android` directory and all of its subdirectories On the other hand, `src/ios` will not be cloned.  This input accepts one path per line, separate entries by a linebreak. |  |  |
| `repository_url` | SSH or HTTPS URL of the repository to clone | required | `$GIT_REPOSITORY_URL` |
| `commit` | Commit SHA to checkout |  | `$BITRISE_GIT_COMMIT` |
| `tag` | Git tag to checkout |  | `$BITRISE_GIT_TAG` |
| `branch` | Git branch to checkout |  | `$BITRISE_GIT_BRANCH` |
| `branch_dest` | The branch that the pull request targets, such as `main` |  | `$BITRISEIO_GIT_BRANCH_DEST` |
| `pull_request_repository_url` | URL of the source repository of a pull request.  This points to the fork repository in builds triggered by pull requests. |  | `$BITRISEIO_PULL_REQUEST_REPOSITORY_URL` |
| `pull_request_merge_branch` | Git ref pointing to the result of merging the PR branch into the destination branch. Even if the source of the PR is a fork, this is a reference to the destination repository.  Example: `refs/pull/14/merge`  Note: not all Git services provide this value. |  | `$BITRISEIO_PULL_REQUEST_MERGE_BRANCH` |
| `pull_request_head_branch` | Git ref pointing to the head of the PR branch. Even if the source of the PR is a fork, this is a reference to the destination repository.  Example: `refs/pull/14/head`  Note: not all Git services provide this value. |  | `$BITRISEIO_PULL_REQUEST_HEAD_BRANCH` |
| `reset_repository` | Reset repository contents with `git reset --hard HEAD` and `git clean -f` before fetching. |  | `No` |
| `build_url` | Unique build URL of this build on Bitrise.io |  | `$BITRISE_BUILD_URL` |
| `build_api_token` | The build's API Token for the build on Bitrise.io | sensitive | `$BITRISE_BUILD_API_TOKEN` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `GIT_CLONE_COMMIT_HASH` | SHA hash of the checked-out commit. |
| `GIT_CLONE_COMMIT_MESSAGE_SUBJECT` | Commit message of the checked-out commit. |
| `GIT_CLONE_COMMIT_MESSAGE_BODY` | Commit message body of the checked-out commit. |
| `GIT_CLONE_COMMIT_COUNT` | Commit count after checkout.  Count will only work properly if no `--depth` option is set. If `--depth` is set then the history truncated to the specified number of commits. Count will **not** fail but will be the clone depth. |
| `GIT_CLONE_COMMIT_AUTHOR_NAME` | Author of the checked-out commit. |
| `GIT_CLONE_COMMIT_AUTHOR_EMAIL` | Email of the checked-out commit. |
| `GIT_CLONE_COMMIT_COMMITTER_NAME` | Committer name of the checked-out commit. |
| `GIT_CLONE_COMMIT_COMMITTER_EMAIL` | Email of the checked-out commit. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-git-clone/pulls) and [issues](https://github.com/bitrise-steplib/steps-git-clone/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
