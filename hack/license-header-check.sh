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

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../" && pwd)"

MISSING_LICENSE_FILES=()

# Define the essential Apache 2.0 license lines
LICENSE_LINES=(
    "http://www.apache.org/licenses/LICENSE-2.0"
)

# Find all .go and .sh files, excluding specified paths
mapfile -t FILES < <(find "$ROOT_DIR" \
    -type f \( -name "*.go" -o -name "*.sh" \) \
    ! -path "$ROOT_DIR/vendor/*" \
    ! -path "$ROOT_DIR/kubevirtci/*" \
    ! -path "$ROOT_DIR/tools/*" \
    ! -path "$ROOT_DIR/staging/*" \
    ! -path "$ROOT_DIR/.bazeldnf/*" \
    ! -path "$ROOT_DIR/pkg/handler-launcher-com/*" \
    ! -path "$ROOT_DIR/pkg/hooks/*" \
    ! -path "$ROOT_DIR/pkg/virt-handler/device-manager/deviceplugin/v1beta1/api.pb.go" \
    ! -path "$ROOT_DIR/pkg/vsock/system/v1/system.pb.go" \
    ! -name "generated_mock*.go")

for file in "${FILES[@]}"; do
    extension="${file##*.}"
    content=$(<"$file")

    if [[ "$extension" == "sh" && "$content" =~ ^#!.*$'\n' ]]; then
        content="${content#*$'\n'}"
    fi

    missing=0
    for line in "${LICENSE_LINES[@]}"; do
        if ! grep -Fq "$line" <<<"$content"; then
            missing=1
            break
        fi
    done

    if [[ "$missing" -eq 1 ]]; then
        MISSING_LICENSE_FILES+=("$file")
    fi
done

if [[ ${#MISSING_LICENSE_FILES[@]} -gt 0 ]]; then
    echo "The following files are missing the required license header:"
    printf '%s\n' "${MISSING_LICENSE_FILES[@]}"
    echo
    echo "Refer to the $ROOT_DIR/LICENSE for guidance on applying the Apache License."
    exit 1
fi

exit 0
