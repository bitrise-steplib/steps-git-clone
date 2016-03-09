#!/bin/bash

#
# NOTE:
#  the raw-ssh-key parameter is a multiline input -> will be directly retrieved from the environment
#

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

formatted_output_file_path=''
if [ -n "${BITRISE_STEP_FORMATTED_OUTPUT_FILE_PATH}" ] ; then
	formatted_output_file_path="${BITRISE_STEP_FORMATTED_OUTPUT_FILE_PATH}"
fi

ruby "${THIS_SCRIPT_DIR}/git_clone.rb" \
	--repo-url="${repository_url}" \
	--commit-hash="${commit}" \
	--tag="$(printf '%q' "${tag}")" \
	--branch="$(printf '%q' "${branch}")" \
	--pull-request="${pull_request_id}" \
	--dest-dir="${clone_into_dir}" \
	--formatted-output-file="${formatted_output_file_path}"
exit $?
