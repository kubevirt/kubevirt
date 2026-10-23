#!/usr/bin/env bash
# Build selected KubeVirt component image targets and print their content digests.
set -eo pipefail

component_list=${KUBEVIRT_FAST_COMPONENTS:?KUBEVIRT_FAST_COMPONENTS is required}
component_list=${component_list//,/ }
read -r -a components <<< "$component_list"
read -r -a targets <<< "${KUBEVIRT_FAST_TARGETS:?KUBEVIRT_FAST_TARGETS is required}"
read -r -a digest_files <<< "${KUBEVIRT_FAST_DIGEST_FILES:?KUBEVIRT_FAST_DIGEST_FILES is required}"
if ((${#targets[@]} != ${#components[@]} || ${#digest_files[@]} != ${#components[@]})); then
    echo "Component, target, and digest-file counts do not match" >&2
    exit 1
fi

# shellcheck disable=SC1091
source hack/common.sh
# shellcheck disable=SC1091
source hack/bootstrap.sh
# shellcheck disable=SC1091
source hack/config.sh

read -r -a bazel_config <<< "${BAZEL_CS_CONFIG:-}"
bazel build --config="${ARCHITECTURE}" "${bazel_config[@]}" \
    --define container_prefix= --define image_prefix= --define container_tag= \
    "${targets[@]}"

for index in "${!components[@]}"; do
    component=${components[$index]}
    digest_file=${digest_files[$index]}
    if [[ ! -f "$digest_file" ]]; then
        echo "No Bazel image digest output found for $component ($digest_file)" >&2
        exit 1
    fi
    digest=$(sha256sum "$digest_file" | awk '{print $1}')
    printf 'FAST_SYNC_DIGEST=%s:%s\n' "$component" "$digest"
done
