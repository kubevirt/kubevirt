#!/usr/bin/env bash

set -exo pipefail

GITHUB_FQDN=github.com

TARGET_BRANCH="$PULL_BASE_REF"
if [ -n "${DOCKER_TAG}" ]; then
    TARGET_TAG="$DOCKER_TAG"
fi

# if we are not on default branch and there is no tag, do nothing
if [ -z "${TARGET_TAG}" ] && [ "${TARGET_BRANCH}" != "main" ]; then
    echo "not on a tag and not on main branch, nothing to do."
    exit 0
fi

function prepare_repo() {
    NAME=$1
    API_REF_DIR=/tmp/${NAME}
    GO_API_REF_REPO=kubevirt/${NAME}
    STAGING_PATH=staging/src/kubevirt.io/${NAME}/
    rm -rf ${API_REF_DIR}
    git clone \
        "https://${GIT_USER_NAME}@${GITHUB_FQDN}/${GO_API_REF_REPO}.git" \
        "${API_REF_DIR}" >/dev/null 2>&1
    pushd ${API_REF_DIR}
    git checkout -B ${TARGET_BRANCH}-local
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
    git config user.email "${GIT_AUTHOR_NAME:-kubevirt-bot}"
    git config user.name "${GIT_AUTHOR_EMAIL:-rmohr+kubebot@redhat.com}"

    git add -A

    if [ -n "$(git status --porcelain)" ]; then
        git commit --message "${NAME} update by KubeVirt Prow build ${BUILD_ID}"
    else
        echo "${NAME} hasn't changed."
    fi
    popd
}

function push_repo() {
    NAME=$1
    API_REF_DIR=/tmp/${NAME}
    pushd ${API_REF_DIR}
    if [ -n "${TARGET_TAG}" ]; then
        if [ $(git tag -l "${TARGET_TAG}") ]; then
            # tag already exists
            echo "tag already exists remotely, doing nothing."
            exit 0
        fi
        git tag ${TARGET_TAG}
        git push origin ${TARGET_TAG}
        echo "${NAME} updated for tag ${TARGET_TAG}."
    else
        if [ "${TARGET_BRANCH}" == "main" ]; then
            git push origin ${TARGET_BRANCH}-local:${TARGET_BRANCH}
            echo "${NAME} updated for ${TARGET_BRANCH}."
        fi
    fi
    popd
}

prepare_repo client-go
commit_repo client-go
push_repo client-go
