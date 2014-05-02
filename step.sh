#!/bin/bash

ruby ./git_clone.rb \
  --repo-url=$GIT_REPOSITORY_URL \
  --branch=$CONCRETE_GIT_BRANCH \
  --dest-dir=$CONCRETE_SOURCE_DIR \
  --auth-username=$AUTH_USER \
  --auth-password=$AUTH_PASSWORD \
  --auth-ssh-base64=$AUTH_SSH_PRIVATE_KEY_BASE64

exit $?