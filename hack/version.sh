#!/bin/bash
#
# Copyright 2014 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#
# Used https://github.com/kubernetes/kubernetes/blob/master/hack/lib/version.sh as a template

# -----------------------------------------------------------------------------
# Version management helpers.  These functions help to set, save and load the
# following variables:
#
#    KUBEVIRT_GIT_COMMIT - The git commit id corresponding to this
#          source code.
#    KUBEVIRT_GIT_TREE_STATE - "clean" indicates no changes since the git commit id
#        "dirty" indicates source code changes after the git commit id
#        "archive" indicates the tree was produced by 'git archive'
#    KUBEVIRT_GIT_VERSION - "vX.Y" used to indicate the last release version.
#    KUBEVIRT_SOURCE_DATE_EPOCH - unix timestamp.  Set to ' ' to generate using
#          'date' instead of 'git'.
#

# Grovels through git to set a set of env variables.

function kubevirt::version::get_version_vars() {
    # If the kubernetes source was exported through git archive, then
    # we likely don't have a git tree, but these magic values may be filled in.
    if [[ '$Format:%%$' == "%" ]]; then
        KUBE_GIT_COMMIT='$Format:%H$'
        KUBE_GIT_TREE_STATE="archive"
        # When a 'git archive' is exported, the '$Format:%D$' below will look
        # something like 'HEAD -> release-1.8, tag: v1.8.3' where then 'tag: '
        # can be extracted from it.
        if [[ '$Format:%D$' =~ tag:\ (v[^ ,]+) ]]; then
            KUBE_GIT_VERSION="${BASH_REMATCH[1]}"
        fi
    fi

    local git=(git --work-tree "${KUBEVIRT_DIR}")

    if [[ -n ${KUBEVIRT_GIT_COMMIT-} ]] || KUBEVIRT_GIT_COMMIT=$("${git[@]}" rev-parse "HEAD^{commit}" 2>/dev/null); then
        if [[ -z ${KUBEVIRT_GIT_TREE_STATE-} ]]; then
            # Check if the tree is dirty.  default to dirty
            if git_status=$("${git[@]}" status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
                KUBEVIRT_GIT_TREE_STATE="clean"
            else
                KUBEVIRT_GIT_TREE_STATE="dirty"
            fi
        fi

        # Use git describe to find the version based on tags.
        if [[ -n ${KUBEVIRT_GIT_VERSION-} ]] || KUBEVIRT_GIT_VERSION=$("${git[@]}" describe --match='v[0-9]*' --tags --abbrev=14 "${KUBEVIRT_GIT_COMMIT}^{commit}" 2>/dev/null); then
            # This translates the "git describe" to an actual semver.org
            # compatible semantic version that looks something like this:
            #   v1.1.0-alpha.0.6+84c76d1142ea4d
            #
            # TODO: We continue calling this "git version" because so many
            # downstream consumers are expecting it there.
            DASHES_IN_VERSION=$(echo "${KUBEVIRT_GIT_VERSION}" | sed "s/[^-]//g")
            if [[ "${DASHES_IN_VERSION}" == "---" ]]; then
                # We have distance to subversion (v1.1.0-subversion-1-gCommitHash)
                KUBEVIRT_GIT_VERSION=$(echo "${KUBEVIRT_GIT_VERSION}" | sed "s/-\([0-9]\{1,\}\)-g\([0-9a-f]\{14\}\)$/.\1\+\2/")
            elif [[ "${DASHES_IN_VERSION}" == "--" ]]; then
                # We have distance to base tag (v1.1.0-1-gCommitHash)
                KUBEVIRT_GIT_VERSION=$(echo "${KUBEVIRT_GIT_VERSION}" | sed "s/-g\([0-9a-f]\{14\}\)$/+\1/")
            fi
            if [[ "${KUBEVIRT_GIT_TREE_STATE}" == "dirty" ]]; then
                # git describe --dirty only considers changes to existing files, but
                # that is problematic since new untracked .go files affect the build,
                # so use our idea of "dirty" from git status instead.
                KUBEVIRT_GIT_VERSION+="-dirty"
            fi

            # If KUBEVIRT_GIT_VERSION is not a valid Semantic Version, then refuse to build.
            if ! [[ "${KUBEVIRT_GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
                echo "KUBEVIRT_GIT_VERSION should be a valid Semantic Version"
                echo "Please see more details here: https://semver.org"
                exit 1
            fi
        fi
    fi
}

function kubevirt::version::ldflag() {
    local key=${1}
    local val=${2}

    echo "-X kubevirt.io/kubevirt/pkg/version.${key}=${val}"
}

# Prints the value that needs to be passed to the -ldflags parameter of go build
function kubevirt::version::ldflags() {
    kubevirt::version::get_version_vars

    SOURCE_DATE_EPOCH=${KUBEVIRT_SOURCE_DATE_EPOCH-$(git show -s --format=format:%ct HEAD)}

    local buildDate
    [[ -z ${SOURCE_DATE_EPOCH-} ]] || buildDate="--date=@${SOURCE_DATE_EPOCH}"
    local -a ldflags=($(kubevirt::version::ldflag "buildDate" "$(date ${buildDate} -u +'%Y-%m-%dT%H:%M:%SZ')"))
    if [[ -n ${KUBEVIRT_GIT_COMMIT-} ]]; then
        ldflags+=($(kubevirt::version::ldflag "gitCommit" "${KUBEVIRT_GIT_COMMIT}"))
        ldflags+=($(kubevirt::version::ldflag "gitTreeState" "${KUBEVIRT_GIT_TREE_STATE}"))
    fi

    if [[ -n ${KUBEVIRT_GIT_VERSION-} ]]; then
        ldflags+=($(kubevirt::version::ldflag "gitVersion" "${KUBEVIRT_GIT_VERSION}"))
    fi

    # The -ldflags parameter takes a single string, so join the output.
    echo "${ldflags[*]-}"
}
