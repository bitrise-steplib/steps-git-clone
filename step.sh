#!/bin/bash

# if [ ! -n "$GIT_BRANCH" ]; then
#   export GIT_BRANCH_PARAMETER="--no-single-branch"
# else
#   export GIT_BRANCH_PARAMETER="--single-branch --branch $GIT_BRANCH"
# fi

# ssh -o StrictHostKeyChecking=no bitbucket.org/github.com -> to add host to known_hosts

# Clone the repository
# git clone --recursive $GIT_BRANCH_PARAMETER $GIT_REPOSITORY_URL $CONCRETE_RELATIVE_GIT_DIRECTORY

ruby ./git_clone.rb