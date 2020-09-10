#!/bin/bash

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
