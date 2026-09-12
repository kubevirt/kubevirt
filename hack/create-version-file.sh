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
# Copyright 2026 The KubeVirt Authors
#
# Create a version file for container images
# This copies what Bazel does with get-version

set -e

VERSION_FILE=${1:-/workspace/.version}

if git rev-parse --git-dir >/dev/null 2>&1; then
    GIT_VERSION=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
else
    GIT_VERSION="unknown"
fi

# Create the version file
echo "${GIT_VERSION}" >"${VERSION_FILE}"
chmod 755 "${VERSION_FILE}"

echo "Created ${VERSION_FILE} with version: ${GIT_VERSION}"
