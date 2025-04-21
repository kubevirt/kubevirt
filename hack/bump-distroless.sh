#!/usr/bin/env bash
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

set -ex -o pipefail

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

DISTROLESS_AMD64_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base:latest-amd64 | jq '.Digest' -r)
DISTROLESS_ARM64_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base:latest-arm64 | jq '.Digest' -r)
DISTROLESS_S390X_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base:latest-s390x | jq '.Digest' -r)

# buildozer returns a non-zero exit code (3) if the commands were a success but did not change the file.
# To make the command idempotent, first set the digest to a kind-of-unique number to work around this behaviour.
# This way we can assume a zero exit code if no errors occur.
cat <<EOF | bazel run --config=${ARCHITECTURE} -- :buildozer -f -
set digest "$(date +%s)"|${KUBEVIRT_DIR}/WORKSPACE:go_image_base_aarch64
set digest "$(date +%s)"|${KUBEVIRT_DIR}/WORKSPACE:go_image_base_aarch64
EOF
# now set the actual digest
cat <<EOF | bazel run --config=${ARCHITECTURE} -- :buildozer -f -
set digest "${DISTROLESS_AMD64_DIGEST}"|${KUBEVIRT_DIR}/WORKSPACE:go_image_base
set digest "${DISTROLESS_ARM64_DIGEST}"|${KUBEVIRT_DIR}/WORKSPACE:go_image_base_aarch64
set digest "${DISTROLESS_S390X_DIGEST}"|${KUBEVIRT_DIR}/WORKSPACE:go_image_base_s390x
EOF
