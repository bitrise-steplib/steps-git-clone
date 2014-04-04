# Concrete
## Git-clone
Clones a specified git repository to the desired relative path

## Environment variables
GIT_REPOSITORY_URL - the git repository you want to clone

GIT_BRANCH - optional; Set only if you want to clone just that branch

## Requirements by Concrete
Before starting this step the core-step-runner should copy the user's private ssh key to the ssh key storage ( ~/.ssh/ ).