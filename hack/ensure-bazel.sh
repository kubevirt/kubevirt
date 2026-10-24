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

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
expected_version="$(tr -d '[:space:]' <"${ROOT_DIR}/.bazelversion")"

ensure_bazel() {
    local current_version
    current_version="$(bazel --version 2>/dev/null | awk '{print $2}' || true)"
    if [[ "${current_version}" == "${expected_version}" ]]; then
        return 0
    fi

    local bazel_arch
    case "$(uname -m)" in
    x86_64 | amd64)
        bazel_arch="x86_64"
        ;;
    aarch64 | arm64)
        bazel_arch="arm64"
        ;;
    s390x)
        return 0
        ;;
    *)
        echo "Unsupported architecture $(uname -m) for automatic Bazel bootstrap" >&2
        return 1
        ;;
    esac

    local target="/usr/bin/bazel"
    if [[ ! -w "/usr/bin" && ! -w "/usr/bin/bazel" ]]; then
        mkdir -p "${HOME}/.local/bin"
        target="${HOME}/.local/bin/bazel"
    fi

    local url="https://github.com/bazelbuild/bazel/releases/download/${expected_version}/bazel-${expected_version}-linux-${bazel_arch}"
    echo "Bootstrapping Bazel ${expected_version} (${bazel_arch}) into ${target} ..." >&2
    local tmp_file="${target}.tmp.$$"
    trap 'rm -f "${tmp_file}"' EXIT
    curl -fsSL --retry 3 -o "${tmp_file}" "${url}"
    chmod 0755 "${tmp_file}"
    mv -f "${tmp_file}" "${target}"
    trap - EXIT
}

if command -v flock >/dev/null 2>&1; then
    exec 9>/tmp/kubevirt-ensure-bazel.lock
    flock 9
    ensure_bazel
    exec 9>&-
else
    ensure_bazel
fi
