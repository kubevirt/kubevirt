#!/usr/bin/env bash

set -ex -o pipefail

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

DISTROLESS_AMD64_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base-debian12:latest-amd64 | jq '.Digest' -r)
DISTROLESS_ARM64_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base-debian12:latest-arm64 | jq '.Digest' -r)
DISTROLESS_S390X_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base-debian12:latest-s390x | jq '.Digest' -r)

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

# Update hack/build-images-container.sh to keep digests in sync
sed -i "/amd64)/,/;;/ s|DISTROLESS_DIGEST=\"sha256:[a-f0-9]*\"|DISTROLESS_DIGEST=\"${DISTROLESS_AMD64_DIGEST}\"|" hack/build-images-container.sh
sed -i "/arm64)/,/;;/ s|DISTROLESS_DIGEST=\"sha256:[a-f0-9]*\"|DISTROLESS_DIGEST=\"${DISTROLESS_ARM64_DIGEST}\"|" hack/build-images-container.sh
sed -i "/s390x)/,/;;/ s|DISTROLESS_DIGEST=\"sha256:[a-f0-9]*\"|DISTROLESS_DIGEST=\"${DISTROLESS_S390X_DIGEST}\"|" hack/build-images-container.sh
