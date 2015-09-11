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

echo " (i) formatted_output_file_path: ${formatted_output_file_path}"

if [ -z "${clone_into_dir}" ] ; then
	echo " [!] No clone_into_dir specified!"
	exit 1
fi
# make clone_into_dir abs
cd "${clone_into_dir}"
clone_into_dir="$(pwd)"

# cd into another dir, in case clone_into_dir would be the current dir
#   because it'll be deleted before git clone
cd "${HOME}"

ruby "${THIS_SCRIPT_DIR}/git_clone.rb" \
	--repo-url="${repository_url}" \
	--commit-hash="${commit}" \
	--tag="${tag}" \
	--branch="${branch}" \
	--pull-request="${pull_request_id}" \
	--dest-dir="${clone_into_dir}" \
	--formatted-output-file="${formatted_output_file_path}" \
	--is-export-outputs="true"
exit $?
