# Concrete
## Git-clone
Clones a specified git repository to the desired relative path

## Environment variables
GIT_REPOSITORY_URL - the git repository you want to clone
AUTH_USER - username for authorizing git repository
AUTH_PASSWORD - password for authorizing git repository
AUTH_SSH_PRIVATE_KEY - private key for authorizing git repository; should be encoded in base64 format

# Notes
- GIT_ASKPASS=echo git clone... -> GIT_ASKPASS=echo will automatically fail if git would show a password prompt


# How-Tos
- how to convert a file into Base64 on OSX: http://superuser.com/a/120815


# Best Practices
- create a new user which can access the repository you want to clone
-- and use this "bot" user's username&password or ssh key, _don't_ use your own, especially don't use your own username&password!