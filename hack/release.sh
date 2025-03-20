#!/bin/bash

source common.sh

set -e

fail_if_cri_bin_missing

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

GIT_USER=${GIT_USER:-$(git config user.name)}
GIT_EMAIL=${GIT_EMAIL:-$(git config user.email)}

echo "git user:  $GIT_USER"
echo "git email: $GIT_EMAIL"

${KUBEVIRT_CRI} pull quay.io/kubevirtci/release-tool:latest

echo "${KUBEVIRT_CRI} run -it --rm \
-v ${GPG_PRIVATE_KEY_FILE}:/home/releaser/gpg-private \
-v ${GPG_PASSPHRASE_FILE}:/home/releaser/gpg-passphrase \
-v ${GITHUB_API_TOKEN_FILE}:/home/releaser/github-api-token \
kubevirtci/release-tool:latest \
--org=kubevirt \
--repo=kubevirt \
--git-email \"${GIT_EMAIL}\" \
--git-user \"${GIT_USER}\"
\"$@\""

${KUBEVIRT_CRI} run -it --rm \
    -v ${GPG_PRIVATE_KEY_FILE}:/home/releaser/gpg-private:Z \
    -v ${GPG_PASSPHRASE_FILE}:/home/releaser/gpg-passphrase:Z \
    -v ${GITHUB_API_TOKEN_FILE}:/home/releaser/github-api-token:Z \
    quay.io/kubevirtci/release-tool:latest \
    --org=kubevirt \
    --repo=kubevirt \
    --git-email "${GIT_EMAIL}" \
    --git-user "${GIT_USER}" \
    "$@"
