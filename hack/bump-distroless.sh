#!/usr/bin/env bash

set -ex -o pipefail

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

DISTROLESS_AMD64_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base:latest-amd64 | jq '.Digest' -r)
DISTROLESS_ARM64_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base:latest-arm64 | jq '.Digest' -r)

bazel run \
    --config=${ARCHITECTURE} \
    -- @com_github_bazelbuild_buildtools//buildozer "set digest \"${DISTROLESS_AMD64_DIGEST}\"" ${KUBEVIRT_DIR}/WORKSPACE:go_image_base

bazel run \
    --config=${ARCHITECTURE} \
    -- @com_github_bazelbuild_buildtools//buildozer "set digest \"${DISTROLESS_ARM64_DIGEST}\"" ${KUBEVIRT_DIR}/WORKSPACE:go_image_base_aarch64
