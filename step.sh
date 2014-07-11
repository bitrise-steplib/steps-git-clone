#!/bin/bash

formatted_output_file_path=''
if [ -n "$GIT_CLONE_FORMATTED_OUTPUT_FILE_PATH" ]; then
	formatted_output_file_path="$GIT_CLONE_FORMATTED_OUTPUT_FILE_PATH"
fi

ruby ./git_clone.rb \
  --repo-url=$GIT_REPOSITORY_URL \
  --commit-hash=$BITRISE_GIT_COMMIT \
  --tag=$BITRISE_GIT_TAG \
  --branch=$BITRISE_GIT_BRANCH \
  --dest-dir=$BITRISE_SOURCE_DIR \
  --auth-username=$AUTH_USER \
  --auth-password=$AUTH_PASSWORD \
  --auth-ssh-base64=$AUTH_SSH_PRIVATE_KEY_BASE64 \
  --formatted-output-file="$formatted_output_file_path"

exit $?