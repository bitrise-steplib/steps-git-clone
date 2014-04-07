# Concrete
## Git-clone
Clones a specified git repository to the desired relative path

## Environment variables
GIT_REPOSITORY_URL - the git repository you want to clone

GIT_BRANCH - optional; Set only if you want to clone just that branch

## Requirements by Concrete
Before starting this step the core-step-runner should copy the user's private ssh key to the ssh key storage ( ~/.ssh/ ).


# How-Tos
- how to convert a file into Base64 on OSX: http://superuser.com/a/120815


# Best Practices
- create a new user which can access the repository you want to clone
-- and use this "bot" user's username&password or ssh key, _don't_ use your own, especially don't use your own username&password!