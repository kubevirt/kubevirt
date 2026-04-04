#!/usr/bin/env bash
#
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
# Copyright 2026 The KubeVirt Authors
#
# This script synchronizes the builder image version from hack/dockerized
# to all Containerfiles in the repository.
#
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Extract builder version from hack/dockerized
DOCKERIZED_FILE="${SCRIPT_DIR}/dockerized"

if [[ ! -f "${DOCKERIZED_FILE}" ]]; then
    echo "Error: ${DOCKERIZED_FILE} not found"
    exit 1
fi

# Extract the version from hack/dockerized
BUILDER_VERSION=$(grep '^kubevirt_builder_version=' "${DOCKERIZED_FILE}" | cut -d'"' -f2)

if [[ -z "${BUILDER_VERSION}" ]]; then
    echo "Error: Could not extract builder version from ${DOCKERIZED_FILE}"
    exit 1
fi

echo "Found builder version in hack/dockerized: ${BUILDER_VERSION}"

BUILDER_IMAGE="quay.io/kubevirt/builder:${BUILDER_VERSION}"

echo "Target builder image: ${BUILDER_IMAGE}"

CONTAINERFILES=$(find "${REPO_ROOT}" -name "Containerfile" -type f \
    ! -path "*/vendor/*" \
    ! -path "*/_out/*" \
    ! -path "*/bazel-*/*" |
    sort)

if [[ -z "${CONTAINERFILES}" ]]; then
    echo "Warning: No Containerfiles found in repository"
    exit 0
fi

# Count of updated files
UPDATED_COUNT=0
SKIPPED_COUNT=0

echo "Updating Containerfiles..."

for containerfile in ${CONTAINERFILES}; do
    # Get the current builder image from the Containerfile
    current_builder=$(grep '^ARG BUILDER_IMAGE=' "${containerfile}" 2>/dev/null | head -1 | cut -d'=' -f2 || echo "")

    if [[ -z "${current_builder}" ]]; then
        echo "SKIPPED: ${containerfile} - no BUILDER_IMAGE ARG found"
        SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
        continue
    fi

    # Check if already up to date
    if [[ "${current_builder}" == "${BUILDER_IMAGE}" ]]; then
        echo "OK: ${containerfile}"
        SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
        continue
    fi

    # Update the Containerfile
    sed -i "s|^ARG BUILDER_IMAGE=.*|ARG BUILDER_IMAGE=${BUILDER_IMAGE}|" "${containerfile}"

    echo "UPDATED: ${containerfile}"
    echo "${current_builder} → ${BUILDER_IMAGE}"
    UPDATED_COUNT=$((UPDATED_COUNT + 1))
done

if [[ ${UPDATED_COUNT} -gt 0 ]]; then
    echo "Builder version synchronized successfully!"
    exit 0
else
    echo "All Containerfiles are already up to date."
    exit 0
fi
