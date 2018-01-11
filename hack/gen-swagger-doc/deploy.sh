#!/usr/bin/env bash

set -e

GITHUB_FQDN=github.com
API_REF_REPO=kubevirt-incubator/api-reference
API_REF_DIR=/tmp/api-reference

TARGET_DIR="$TRAVIS_BRANCH"
if [ "$TRAVIS_TAG" -n ] ; then
    TARGET_DIR="$TRAVIS_TAG"
fi

git clone \
    "https://${API_REFERENCE_PUSH_TOKEN}@${GITHUB_FQDN}/${API_REF_REPO}.git" \
    "${API_REF_DIR}" >/dev/null 2>&1
rm -rf "${API_REF_DIR}/${TARGET_DIR}/"*
cp -f _out/apidocs/html/*.html "${API_REF_DIR}/${TARGET_DIR}/"

cd "${API_REF_DIR}"

git config --global user.email "travis@travis-ci.org"
git config --global user.name "Travis CI"

if git status --porcelain | grep --quiet "^ M"; then
    git add -A ${TARGET_DIR}/*.html
    git commit --message "API Reference update by Travis Build ${TRAVIS_BUILD_NUMBER}"

    git push origin master >/dev/null 2>&1
    echo "API Reference updated for ${TARGET_DIR}."
else
    echo "API Reference hasn't changed."
fi
