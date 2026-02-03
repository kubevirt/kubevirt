#!/usr/bin/env bash

set -ex -o pipefail

source hack/common.sh
source hack/config.sh

# Bump distroless base image digests
# This script fetches the latest distroless digests and updates the Containerfiles

DISTROLESS_AMD64_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base-debian12:latest-amd64 | jq '.Digest' -r)
DISTROLESS_ARM64_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base-debian12:latest-arm64 | jq '.Digest' -r)
DISTROLESS_S390X_DIGEST=$(skopeo inspect docker://gcr.io/distroless/base-debian12:latest-s390x | jq '.Digest' -r)

echo "Latest distroless digests:"
echo "  AMD64: ${DISTROLESS_AMD64_DIGEST}"
echo "  ARM64: ${DISTROLESS_ARM64_DIGEST}"
echo "  S390X: ${DISTROLESS_S390X_DIGEST}"

# Update Containerfiles with new digests
# The native build uses Containerfiles in build/ directory
# For now, just print the digests - Containerfiles use tags, not digests
echo ""
echo "Note: Native build Containerfiles use image tags, not digests."
echo "To pin to specific digests, update the FROM lines in build/*/Containerfile"
