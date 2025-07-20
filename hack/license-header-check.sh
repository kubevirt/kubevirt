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
# Copyright the KubeVirt Authors.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../" && pwd)"

TARGET_DIRS=(
    "pkg/monitoring"
    "pkg/instancetype"
    "pkg/network"
    "pkg/storage"
    "pkg/container-disk"
    "pkg/emptydisk"
    "pkg/host-disk"
    "pkg/hotplug-disk"
    "pkg/libdv"
    "pkg/virtiofs"
    "pkg/virt-controller/watch/volume-migration"
    "pkg/virtctl/guestfs"
    "pkg/virtctl/imageupload"
    "pkg/virtctl/memorydump"
    "pkg/virtctl/vmexport"
)

IGNORE_FILES=(
    # Generated files
    "pkg/network/dhcp/generated_mock_configurator.go"
    "pkg/network/driver/generated_mock_common.go"
)

MISSING_LICENSE_FILES=()

# Define the required license lines based on hack/boilerplate (order does not matter)
LICENSE_FILE="$ROOT_DIR/hack/boilerplate/boilerplate.go.txt"
LICENSE_LINES=()

while IFS= read -r line; do
    stripped=$(echo "$line" | sed -E 's/^[[:space:]]*(\/\/|\#|\*|\/\*|\*\/)?[[:space:]]*//')
    [[ -n "$stripped" ]] && LICENSE_LINES+=("$stripped")
done <"$LICENSE_FILE"

FILES=()
for dir in "${TARGET_DIRS[@]}"; do
    while IFS= read -r -d '' file; do
        FILES+=("${file#$ROOT_DIR/}")
    done < <(find "$ROOT_DIR/$dir" -type f \( -name "*.go" -o -name "*.sh" \) -print0)
done

for rel_file in "${FILES[@]}"; do
    for ignore in "${IGNORE_FILES[@]}"; do
        if [[ "$rel_file" == "$ignore" ]]; then
            continue 2
        fi
    done

    file="$ROOT_DIR/$rel_file"
    extension="${file##*.}"

    content=$(head -n 40 "$file" | sed -E 's/^[[:space:]]*(\/\/|\#|\*)[[:space:]]*//; s|^/\*+||; s|\*/$||')
    normalized_content=$(echo "$content" | sed 's/^[[:space:]]*//')

    missing_line=0
    for license_line in "${LICENSE_LINES[@]}"; do
        if ! grep -Fq "$license_line" <<<"$normalized_content"; then
            missing_line=1
            break
        fi
    done

    if [[ "$missing_line" -eq 1 ]]; then
        MISSING_LICENSE_FILES+=("$rel_file")
    fi
done

if [[ ${#MISSING_LICENSE_FILES[@]} -gt 0 ]]; then
    echo "The following files are missing the required license header:"
    printf '%s\n' "${MISSING_LICENSE_FILES[@]}"
    echo
    echo "Refer to the README file for guidance on applying the Apache License."
    exit 1
fi

exit 0
