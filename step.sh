#!/bin/bash

#
# NOTE:
#  the raw-ssh-key parameter is a multiline input -> will be directly retrieved from the environment
#

formatted_output_file_path=''
if [ -n "${GIT_CLONE_FORMATTED_OUTPUT_FILE_PATH}" ] ; then
	formatted_output_file_path="${GIT_CLONE_FORMATTED_OUTPUT_FILE_PATH}"
fi

echo " (i) formatted_output_file_path: ${formatted_output_file_path}"

is_export_outputs='false'
if [[ "${GIT_CLONE_IS_EXPORT_OUTPUTS}" == "true" ]] ; then
	is_export_outputs='true'
fi

echo " (i) is_export_outputs: ${is_export_outputs}"

ruby ./git_clone.rb \
	--repo-url="${GIT_REPOSITORY_URL}" \
	--commit-hash="${BITRISE_GIT_COMMIT}" \
	--tag="${BITRISE_GIT_TAG}" \
	--branch="${BITRISE_GIT_BRANCH}" \
	--dest-dir="${BITRISE_SOURCE_DIR}" \
	--auth-username="${AUTH_USER}" \
	--auth-password="${AUTH_PASSWORD}" \
	--auth-ssh-base64="${AUTH_SSH_PRIVATE_KEY_BASE64}" \
	--formatted-output-file="${formatted_output_file_path}" \
	--is-export-outputs="${is_export_outputs}"

exit $?