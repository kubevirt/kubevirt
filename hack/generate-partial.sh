#!/usr/bin/env bash
#
# Incremental variant of hack/generate.sh, called by hack/generate-incremental.sh
# when the only changes since the last successful `make generate` are under
# pkg/, cmd/ or tests/ (outside the paths that feed protoc or the sync scripts).
#
# Such changes cannot affect the API code generators (swagger-doc, deepcopy-gen,
# client-gen, openapi-gen, controller-gen, ...) or protoc, so those are skipped.
# What still runs:
#   - go generate, only in the changed directories that contain directives
#   - deepcopy-gen for pkg/virt-launcher/virtwrap/api, if it changed
#   - ginkgo suite bootstrapping
#   - the generator tools (manifests, CSV, OpenAPI spec, docs, examples)
#
# hack/generate-incremental.sh runs bazel-fmt.sh and bazel-race.sh afterwards.
#
# Usage: hack/generate-partial.sh <file with one changed directory per line>

set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

changed_dirs_file=$1
if [ -z "${changed_dirs_file}" ] || [ ! -f "${changed_dirs_file}" ]; then
    echo "usage: $0 <changed-dirs-file>" >&2
    exit 1
fi

virtwrap_api_dir=pkg/virt-launcher/virtwrap/api

while IFS= read -r dir; do
    # The directory may have been deleted since the last run.
    [ -d "${KUBEVIRT_DIR}/${dir}" ] || continue

    # hack/generate.sh only runs go generate for pkg/, staging/ and tests/;
    # staging/ changes always take the full path.
    has_directives=false
    if [[ ${dir} == pkg/* || ${dir} == tests/* ]] &&
        grep -qs '^//go:generate' "${KUBEVIRT_DIR}/${dir}"/*.go; then
        has_directives=true
    fi

    if [ "${has_directives}" = false ] && [ "${dir}" != "${virtwrap_api_dir}" ]; then
        continue
    fi

    # Same cleanup hack/generate.sh does for all of pkg/, scoped to this
    # directory, so that removed directives do not leave stale files behind.
    if [[ ${dir} == pkg/* ]]; then
        find "${KUBEVIRT_DIR}/${dir}" -maxdepth 1 -name "*generated*.go" -delete
    fi

    if [ "${has_directives}" = true ]; then
        go generate -tags "${KUBEVIRT_GO_BUILD_TAGS}" "./${dir}"
    fi

    if [ "${dir}" = "${virtwrap_api_dir}" ]; then
        deepcopy-gen \
            --output-file deepcopy_generated.go \
            --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt \
            ./${virtwrap_api_dir}
    fi
done <"${changed_dirs_file}"

${KUBEVIRT_DIR}/hack/bootstrap-ginkgo.sh

${KUBEVIRT_DIR}/hack/generate-tools.sh
