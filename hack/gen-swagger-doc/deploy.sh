#!/usr/bin/env bash

set -e

GITHUB_FQDN=github.com
API_REF_REPO=kubevirt-incubator/api-reference
API_REF_DIR=/tmp/api-reference

git clone \
    "https://${API_REFERENCE_PUSH_TOKEN}@${GITHUB_FQDN}/${API_REF_REPO}.git" \
    "${API_REF_DIR}" >/dev/null 2>&1
rm -rf "${API_REF_DIR}/content/"*
cp -f hack/gen-swagger-doc/html5/content/*.html "${API_REF_DIR}/content/"

cd "${API_REF_DIR}"

git config --global user.email "travis@travis-ci.org"
git config --global user.name "Travis CI"

if git status --porcelain | grep --quiet "^ M"; then
    git add -A content/*.html
    git commit --message "API Reference update by Travis Build ${TRAVIS_BUILD_NUMBER}"

    git push origin master >/dev/null 2>&1
    echo "API Reference updated."
else
    echo "API Reference hasn't changed."
fi
