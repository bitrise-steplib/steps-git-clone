#!/bin/bash

#
# NOTE:
#  the raw-ssh-key parameter is a multiline input -> will be directly retrieved from the environment
#

formatted_output_file_path=''
if [ -n "${BITRISE_STEP_FORMATTED_OUTPUT_FILE_PATH}" ] ; then
	formatted_output_file_path="${BITRISE_STEP_FORMATTED_OUTPUT_FILE_PATH}"
fi

echo " (i) formatted_output_file_path: ${formatted_output_file_path}"

is_export_outputs='false'
if [[ "${is_expose_outputs}" == "true" ]] ; then
	is_export_outputs='true'
fi

echo " (i) is_export_outputs: ${is_export_outputs}"

# DEPRECATED:
#  --auth-ssh-base64 : will be removed in the next version
ruby ./git_clone.rb \
	--repo-url="${repository_url}" \
	--commit-hash="${commit}" \
	--tag="${tag}" \
	--branch="${branch}" \
	--pull-request="${pull_request_id}" \
	--dest-dir="${clone_into_dir}" \
	--auth-username="${auth_user}" \
	--auth-password="${auth_password}" \
	--auth-ssh-base64="${AUTH_SSH_PRIVATE_KEY_BASE64}" \
	--formatted-output-file="${formatted_output_file_path}" \
	--is-export-outputs="${is_export_outputs}"
exit $?
