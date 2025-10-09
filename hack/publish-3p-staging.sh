#!/usr/bin/env bash

set -exo pipefail

GITHUB_FQDN=github.com
GITHUB_USER="${GIT_AUTHOR_EMAIL:-deckhouse-BOaTswain@users.noreply.github.com}"
GITHUB_USER_EMAIL="${GIT_AUTHOR_NAME:-deckhouse-BOaTswain}"
THIRD_PARTY_REPO_NAME=$1
KUBEVIRT_API_DIR_NAME=api

function prepare_repo() {
    NAME=$1
    API_REF_DIR=/tmp/${NAME}
    GO_API_REF_REPO=deckhouse/${NAME}
    STAGING_PATH=staging/src/kubevirt.io/${KUBEVIRT_API_DIR_NAME}/
    rm -rf ${API_REF_DIR}
    git clone \
        "https://${GITHUB_TOKEN}@${GITHUB_FQDN}/${GO_API_REF_REPO}.git" \
        "${API_REF_DIR}" >/dev/null 2>&1
    pushd ${API_REF_DIR}
    git checkout -B ${BRANCH_NAME}
    git pull --rebase origin ${BRANCH_NAME} || echo "Branch $BRANCH_NAME does not exist on remote or pull failed."
    git rm -rf .
    git clean -fxd
    popd
    cp -rf ${STAGING_PATH}/. "${API_REF_DIR}/"

    # copy files which are the same on both repos
    cp -f LICENSE "${API_REF_DIR}/"
    cp -f SECURITY.md "${API_REF_DIR}/"

    pushd ${API_REF_DIR}
    # Generate .gitignore file. We want to keep bazel files in kubevirt/kubevirt, but not in sync target repos
    cat >.gitignore <<__EOF__
BUILD
BUILD.bazel
__EOF__
    popd
}

function commit_repo() {
    NAME=$1
    API_REF_DIR=/tmp/${NAME}
    pushd ${API_REF_DIR}
    git config user.email "${GITHUB_USER}"
    git config user.name "${GITHUB_USER_EMAIL}"

    git add -A

    if [ -n "$(git status --porcelain)" ]; then
        git commit --message "${NAME} update by 3p-kubevirt ${COMMIT_HASH}"
    else
        echo "${NAME} hasn't changed."
    fi
    popd
}

function push_tag() {
    NAME=$1

    if [[ "${NAME}" == "3p-kubevirt-api" ]]; then
        API_REF_DIR=/tmp/${NAME}
        pushd ${API_REF_DIR}
    fi
   
    git config user.email "${GITHUB_USER}"
    git config user.name "${GITHUB_USER_EMAIL}"

    if [ -n "${TARGET_TAG}" ]; then
        if [ $(git tag --list "${TARGET_TAG}") ]; then
            # tag already exists
            echo "tag already exists remotely, doing nothing."
            exit 0
        fi
        git tag ${TARGET_TAG}
        git push origin ${TARGET_TAG}
        echo "${NAME} updated for tag ${TARGET_TAG}."
    fi

    if [[ "${NAME}" == "3p-kubevirt-api" ]]; then
        popd
    fi
}

function push_repo() {
    NAME=$1
    API_REF_DIR=/tmp/${NAME}
    pushd ${API_REF_DIR}

    git push origin --set-upstream ${BRANCH_NAME}
    echo "${NAME} updated for ${BRANCH_NAME}."

    popd
}

if [[ "${THIRD_PARTY_REPO_NAME}" == "3p-kubevirt" ]]; then
    push_tag ${THIRD_PARTY_REPO_NAME}
elif [[ ${THIRD_PARTY_REPO_NAME} == "3p-kubevirt-api" ]]; then
    prepare_repo ${THIRD_PARTY_REPO_NAME}
    commit_repo ${THIRD_PARTY_REPO_NAME}
    push_repo ${THIRD_PARTY_REPO_NAME}
    push_tag ${THIRD_PARTY_REPO_NAME}
else
    echo "Unknown repository name: ${THIRD_PARTY_REPO_NAME}."
    exit 1
fi
