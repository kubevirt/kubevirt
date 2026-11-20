#!/usr/bin/env bash

set -eo pipefail

source hack/common.sh

SBOM_OUT_DIR="${SBOM_OUT_DIR:-${OUT_DIR}/sboms}"
SBOM_FILE="${SBOM_FILE:-${SBOM_OUT_DIR}/kubevirt-source.spdx.json}"
SBOM_ARCHIVE="${SBOM_ARCHIVE:-${SBOM_OUT_DIR}/kubevirt-sbom.tar.gz}"

if ! command -v syft >/dev/null 2>&1; then
    echo "syft is required to generate the release SBOM" >&2
    exit 1
fi

mkdir -p "${SBOM_OUT_DIR}"
rm -f "${SBOM_FILE}" "${SBOM_ARCHIVE}"

syft "dir:${KUBEVIRT_DIR}" -o "spdx-json=${SBOM_FILE}"

tar -czf "${SBOM_ARCHIVE}" -C "${SBOM_OUT_DIR}" "$(basename "${SBOM_FILE}")"

echo "Generated ${SBOM_FILE}"
echo "Generated ${SBOM_ARCHIVE}"
