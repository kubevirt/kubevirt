#!/usr/bin/env bash
#
# Entry point for `make generate`. With KUBEVIRT_GENERATE_INCREMENTAL=true it
# skips the parts of the generate workflow whose inputs have not changed since
# the last successful run. Without it, the full workflow always runs, as before;
# the snapshot below is still recorded, so the first incremental run can
# already skip.
#
# How it works:
#   After every successful run we store a git tree hash of the whole working
#   tree (tracked + untracked, non-ignored files), taken *after* generation, so
#   it includes the generated files. On the next run we take a new snapshot and
#   diff the two:
#     - no differences            -> skip (no container is started)
#     - only pkg/, cmd/, tests/   -> partial run (hack/generate-partial.sh)
#       or *.md changes
#     - anything else             -> full run (the original workflow)
#   Because the snapshot covers the whole tree, newly added generated files are
#   picked up automatically, and deleting or editing a generated file triggers
#   regeneration.
#
# A full run is always done when:
#   - KUBEVIRT_GENERATE_INCREMENTAL is not set to true
#   - CI=true or KUBEVIRT_GENERATE_FULL=true (generate-verify sets this)
#   - there is no stored snapshot, or git has garbage-collected it
#   - environment variables affecting the output, or the ignored local config
#     files (hack/config-local.sh, hack/config-provider-*), changed

set -e

source hack/common.sh

cd "${KUBEVIRT_DIR}"

STATE_FILE="$(git rev-parse --git-path kubevirt-generate-state)"
CHANGED_DIRS_FILE=".generate-changed-dirs"

# Hash the current working tree. Starting from a copy of the real index lets
# git skip re-hashing files whose stat info did not change, which keeps this
# fast even with a large vendor/ directory.
snapshot_tree() {
    local tmp_index
    tmp_index="$(mktemp -u)"
    if [ -f "$(git rev-parse --git-path index)" ]; then
        cp "$(git rev-parse --git-path index)" "${tmp_index}"
    fi
    GIT_INDEX_FILE="${tmp_index}" git add -A . >/dev/null
    GIT_INDEX_FILE="${tmp_index}" git write-tree
    rm -f "${tmp_index}"
}

# Hash the inputs that are not part of the git tree.
env_fingerprint() {
    {
        echo "DOCKER_PREFIX=${DOCKER_PREFIX}"
        echo "DOCKER_TAG=${DOCKER_TAG}"
        echo "IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY}"
        echo "VERBOSITY=${VERBOSITY}"
        echo "KUBEVIRT_CENTOS_STREAM_VERSION=${KUBEVIRT_CENTOS_STREAM_VERSION}"
        echo "KUBEVIRT_BUILDER_IMAGE=${KUBEVIRT_BUILDER_IMAGE}"
        echo "KUBEVIRT_NO_BAZEL=${KUBEVIRT_NO_BAZEL}"
        echo "KUBEVIRT_RUN_UNNESTED=${KUBEVIRT_RUN_UNNESTED}"
        cat hack/config-local.sh hack/config-provider-* 2>/dev/null || true
    } | sha256sum | cut -d' ' -f1
}

generate_env() {
    echo "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY}"
}

# bazel-race.sh sets `race = "on"` on go_test rules. It has to run again after
# gazelle, because gazelle creates the BUILD files of new packages: without
# this, a new package would only get `race = "on"` on the *next* run, and with
# caching that next run is skipped.
run_full() {
    hack/dockerized hack/build-ginkgo.sh
    hack/dockerized "$(generate_env) ./hack/generate.sh"
    SYNC_VENDOR=true hack/dockerized "./hack/bazel-generate.sh && hack/bazel-fmt.sh && ./hack/bazel-race.sh"
    hack/dockerized hack/sync-kubevirtci.sh
    hack/dockerized hack/common-instancetypes/sync.sh
    hack/dockerized hack/virt-template/sync.sh
    ./hack/update-generated-api-testdata.sh
}

run_partial() {
    if [ ! -x "${OUT_DIR}/tests/ginkgo" ]; then
        hack/dockerized hack/build-ginkgo.sh
    fi
    # One container call: generate-partial.sh, formatting, then bazel-race.
    # bazel-fmt.sh runs gazelle on pkg/ tools/ tests/ cmd/, which covers these
    # changes; the repo-wide gazelle in bazel-generate.sh is only needed for
    # vendor/.
    hack/dockerized "$(generate_env) ./hack/generate-partial.sh ${CHANGED_DIRS_FILE} && hack/bazel-fmt.sh && ./hack/bazel-race.sh"
}

record_state() {
    echo "$(snapshot_tree) $(env_fingerprint)" >"${STATE_FILE}"
}

# Classify the changes between two snapshots. Sets `mode` to skip, partial or
# full, and writes the changed directories for a partial run to
# CHANGED_DIRS_FILE.
classify_changes() {
    local old=$1 new=$2 path dirs=""
    mode=skip

    while IFS= read -r path; do
        case "${path}" in
        # Inputs of protoc, outputs of the sync scripts, and docs/metrics.md.
        *.proto | pkg/handler-launcher-com/* | \
            pkg/virt-operator/resource/generate/components/data/* | \
            docs/observability/*)
            echo "generate: ${path} changed, running full generate"
            mode=full
            return
            ;;
        pkg/* | cmd/* | tests/*)
            mode=partial
            if [[ ${path} == *.go ]]; then
                dirs="${dirs}$(dirname "${path}")"$'\n'
            fi
            ;;
        *.md)
            # Documentation does not feed any generator.
            ;;
        *)
            echo "generate: ${path} changed, running full generate"
            mode=full
            return
            ;;
        esac
    done < <(git -c core.quotePath=false diff --name-only --no-renames "${old}" "${new}")

    if [ "${mode}" = partial ]; then
        printf '%s' "${dirs}" | LC_ALL=C sort -u >"${CHANGED_DIRS_FILE}"
    fi
}

main() {
    local old_tree="" old_env="" new_tree mode

    if [ -f "${STATE_FILE}" ]; then
        read -r old_tree old_env <"${STATE_FILE}" || true
    fi

    if [ "${KUBEVIRT_GENERATE_INCREMENTAL}" != "true" ]; then
        echo "generate: running full generate (set KUBEVIRT_GENERATE_INCREMENTAL=true to skip unchanged work)"
        mode=full
    elif [ "${CI}" = "true" ] || [ "${KUBEVIRT_GENERATE_FULL}" = "true" ]; then
        echo "generate: full generate requested"
        mode=full
    elif [ -z "${old_tree}" ] || ! git cat-file -e "${old_tree}^{tree}" 2>/dev/null; then
        echo "generate: no previous successful run found, running full generate"
        mode=full
    elif [ "${old_env}" != "$(env_fingerprint)" ]; then
        echo "generate: environment or local config changed, running full generate"
        mode=full
    else
        new_tree="$(snapshot_tree)"
        classify_changes "${old_tree}" "${new_tree}"
    fi

    # If this run fails, the stored snapshot is left as it is: the next run
    # diffs against the last *successful* run, which still covers everything
    # that changed since then, including anything a failed run wrote.
    trap 'rm -f "${CHANGED_DIRS_FILE}"' EXIT

    case "${mode}" in
    skip)
        if [ ! -x "${OUT_DIR}/tests/ginkgo" ]; then
            hack/dockerized hack/build-ginkgo.sh
        fi
        echo "generate: nothing changed since the last successful run, skipping"
        echo "generate: set KUBEVIRT_GENERATE_FULL=true to force a full run"
        ;;
    partial)
        echo "generate: only pkg/, cmd/ or tests/ changed, running partial generate"
        run_partial
        ;;
    full)
        run_full
        ;;
    esac

    rm -f "${CHANGED_DIRS_FILE}"
    record_state
}

main "$@"
