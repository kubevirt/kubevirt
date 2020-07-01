#!/bin/bash

set -e

if [ -z ${GPG_PRIVATE_KEY_FILE} ]; then
	echo "GPG_PRIVATE_KEY_FILE env var must be set"
	exit 1
elif [ -z ${GPG_PASSPHRASE_FILE} ]; then
	echo "GPG_PASSPHRASE_FILE env var must be set"
	exit 1
elif [ -z ${GITHUB_API_TOKEN_FILE} ]; then
	echo "GITHUB_API_TOKEN_FILE env var must be set"
	exit 1
fi

GIT_USER=${GIT_USER:-`git config user.name`}
GIT_EMAIL=${GIT_EMAIL:-`git config user.email`}

docker pull kubevirtci/release-tool:latest

docker run -it --rm \
-v ${GPG_PRIVATE_KEY_FILE}:/home/releaser/gpg-private \
-v ${GPG_PASSPHRASE_FILE}:/home/releaser/gpg-passphrase \
-v ${GITHUB_API_TOKEN_FILE}:/home/releaser/github-api-token \
kubevirtci/release-tool:latest \
--org=kubevirt \
--repo=kubevirt \
--git-email "${GIT_USER}" \
--git-user ${GIT_EMAIL} \
"$@"


