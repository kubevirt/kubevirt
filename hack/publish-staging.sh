#!/usr/bin/env bash

set -ex

GITHUB_FQDN=github.com
API_REF_REPO=kubevirt/client-go
API_REF_DIR=/tmp/client-go
GITHUB_IO_FQDN="https://kubevirt.github.io/client-go"

TARGET_BRANCH="$TRAVIS_BRANCH"
if [ -n "${TRAVIS_TAG}" ]; then
    TARGET_TAG="$TRAVIS_TAG"
fi

# if we are not on master and there is no tag, do nothing
if [ -z "${TARGET_TAG}" ] && [ "${TARGET_BRANCH}" != "master" ]; then
    echo "not on a tag and not on master branch, nothing to do."
    exit 0
fi

rm -rf ${API_REF_DIR}
git clone \
    "https://${API_REFERENCE_PUSH_TOKEN}@${GITHUB_FQDN}/${API_REF_REPO}.git" \
    "${API_REF_DIR}" >/dev/null 2>&1
pushd ${API_REF_DIR}
git checkout -B ${TARGET_BRANCH}-local
git rm -rf .
git clean -fxd
popd
cp -rf staging/src/kubevirt.io/client-go/. "${API_REF_DIR}/"

# copy files which are the same on both repos
cp -f LICENSE "${API_REF_DIR}/"
cp -f SECURITY.md "${API_REF_DIR}/"

cd "${API_REF_DIR}"

# Generate .gitignore file. We want to keep bazel files in kubevirt/kubevirt, but not in kubevirt/client-go
cat >.gitignore <<__EOF__
BUILD
BUILD.bazel
__EOF__

git config --global user.email "travis@travis-ci.org"
git config --global user.name "Travis CI"

git add -A

if [ -n "$(git status --porcelain)" ]; then
    git commit --message "client-go update by Travis Build ${TRAVIS_BUILD_NUMBER}"

    # we only push branch changes on master
    if [ "${TARGET_BRANCH}" == "master" ]; then
        git push origin ${TARGET_BRANCH}
        echo "client-go updated for ${TARGET_BRANCH}."
    fi
else
    echo "client-go hasn't changed."
fi

if [ -n "${TARGET_TAG}" ]; then
    if [ $(git tag -l "${TARGET_TAG}") ]; then
        # tag already exists
        echo "tag already exists remotely, doing nothing."
        exit 0
    fi
    git tag ${TARGET_TAG}
    git push origin ${TARGET_TAG}
    echo "client-go updated for tag ${TARGET_TAG}."
fi
