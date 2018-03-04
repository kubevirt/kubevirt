#!/usr/bin/env bash

set -e

source $(dirname "$0")/../common.sh

GITHUB_FQDN=github.com
CLIENT_PYTHON_REPO=kubevirt/client-python
CLIENT_PYTHON_DIR=/tmp/kubevirt-client-python

# Reusing API_REFERENCE_PUSH_TOKEN.
git clone \
    "https://${API_REFERENCE_PUSH_TOKEN}@${GITHUB_FQDN}/${CLIENT_PYTHON_REPO}.git" \
    "${CLIENT_PYTHON_DIR}" >/dev/null 2>&1

# Remove content under kubevirt, docs and test directories
rm -rf "${CLIENT_PYTHON_DIR}"/{kubevirt,docs,test}
# Copy client-python into repository
cp -rf "${PYTHON_CLIENT_OUT_DIR}"/* "${CLIENT_PYTHON_DIR}/"

cd "${CLIENT_PYTHON_DIR}"

git config --global user.email "travis@travis-ci.org"
git config --global user.name "Travis CI"

# Push only in case something got changed in code.
if git status --porcelain | grep 'kubevirt/' | grep --quiet "^ [AM]"; then
    git add -A .
    git commit --message "Client Python update by Travis Build ${TRAVIS_BUILD_NUMBER}"

    git push origin master >/dev/null 2>&1
    echo "Client Python updated."
else
    echo "Client Python hasn't changed."
fi
