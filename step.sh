#!/bin/bash

# if [ ! -n "$GIT_BRANCH" ]; then
#   export GIT_BRANCH_PARAMETER="--no-single-branch"
# else
#   export GIT_BRANCH_PARAMETER="--single-branch --branch $GIT_BRANCH"
# fi

# ssh -o StrictHostKeyChecking=no bitbucket.org/github.com -> to add host to known_hosts

# Clone the repository
# git clone --recursive $GIT_BRANCH_PARAMETER $GIT_REPOSITORY_URL $CONCRETE_RELATIVE_GIT_DIRECTORY

ruby ./git_clone.rb --repo-url=$GIT_REPOSITORY_URL --branch=$GIT_BRANCH --dest-dir=$CLONE_DESTINATION_DIR --auth-username=$AUTH_USER --auth-password=$AUTH_PASSWORD --auth-ssh-base64=$AUTH_SSH_PRIVATE_KEY_BASE64