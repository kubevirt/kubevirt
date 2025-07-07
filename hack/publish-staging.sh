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
    git config user.email "${GIT_AUTHOR_EMAIL:-kubevirtbot@redhat.com}"
    git config user.name "${GIT_AUTHOR_NAME:-kubevirt-bot}"

    git add -A

    if [ -n "$(git status --porcelain)" ]; then
        git commit --message "${NAME} update by KubeVirt Prow build ${BUILD_ID}"
    else
        echo "${NAME} hasn't changed."
    fi
    popd
}

function get_pseudo_tag() {
    NAME=$1
    API_REF_DIR=/tmp/${NAME}
    pushd ${API_REF_DIR} >/dev/null
    echo "v0.0.0-$(TZ=UTC0 git show --quiet --date='format-local:%Y%m%d%H%M%S' --format="%cd")-$(git rev-parse --short=12 HEAD)"
    popd >/dev/null
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

function go_mod_remove_staging() {
    NAME=$1
    API_REF_DIR=/tmp/${NAME}
    pushd ${API_REF_DIR}
    go mod edit -dropreplace kubevirt.io/api
    popd
}

function go_mod_populate_pseudoversion() {
    NAME=$1
    API_REF_DIR=/tmp/${NAME}
    pushd ${API_REF_DIR}
    local repo=$2
    local version=$3
    go mod edit -require ${repo}@${version}
    popd
}

# prepare kubevirt.io/api
prepare_repo api
commit_repo api
API_PSEUDO_TAG="$(get_pseudo_tag api)"

# prepare kubevirt.io/client-go
prepare_repo client-go
go_mod_remove_staging client-go
go_mod_populate_pseudoversion client-go kubevirt.io/api ${API_PSEUDO_TAG}
commit_repo client-go

# push the prepared repos
push_repo api
push_repo client-go
