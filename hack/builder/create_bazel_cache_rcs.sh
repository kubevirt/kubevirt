#!/bin/bash
# This file is part of the KubeVirt project
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
# Copyright The KubeVirt Authors.
#


# heavily borrowed from https://raw.githubusercontent.com/kubernetes/test-infra/5489305c550e250f7d9053528c725656d5b5ff20/images/bootstrap/create_bazel_cache_rcs.sh

CACHE_HOST="${CACHE_HOST:-bazel-cache.default.svc.cluster.local.}"
CACHE_PORT="${CACHE_PORT:-8080}"

get_workspace() {
    # get org/repo from prow, otherwise use $PWD
    if [[ -n "${REPO_NAME}" ]] && [[ -n "${REPO_OWNER}" ]]; then
        echo "${REPO_OWNER}/${REPO_NAME}"
    else
        echo "$(basename "$(dirname "$PWD")")/$(basename "$PWD")"
    fi
}

make_bazel_rc() {
    # this is the default for recent releases but we set it explicitly
    # since this is the only hash our cache supports
    echo "startup --host_jvm_args=-Dbazel.DigestFunction=sha256"
    # don't fail if the cache is unavailable
    echo "build --remote_local_fallback"
    # point bazel at our http cache ...
    # NOTE our caches are versioned by all path segments up until the last two
    # IE PUT /foo/bar/baz/cas/asdf -> is in cache "/foo/bar/baz"
    local cache_id
    cache_id="$(get_workspace)"
    local cache_url
    cache_url="http://${CACHE_HOST}:${CACHE_PORT}/${cache_id}"
    echo "build --remote_http_cache=${cache_url}"
}

bazel_rc_contents=$(make_bazel_rc)
echo "create_bazel_cache_rcs.sh: Configuring './ci.bazelrc' with"
echo "# ------------------------------------------------------------------------------"
echo "${bazel_rc_contents}"
echo "# ------------------------------------------------------------------------------"
echo "${bazel_rc_contents}" >>"./ci.bazelrc"
