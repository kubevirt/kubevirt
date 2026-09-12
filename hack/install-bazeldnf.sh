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
# Copyright 2025 Red Hat, Inc.
#
# Downloads the standalone bazeldnf binary for the current platform
# and caches it in _out/tools/bazeldnf. Subsequent runs skip the
# download when the cached binary already matches the expected checksum.
#
# Usage:
#   source hack/install-bazeldnf.sh   # sets BAZELDNF on PATH
#   bazeldnf --help
#

set -e

BAZELDNF_VERSION="v0.5.9-2"
BAZELDNF_REPO="brianmcarey/bazeldnf"

function bazeldnf::sha256_for_platform() {
    case "$1" in
        linux-amd64)  echo "e78b730e5f9d1edeb7b54e7414e8c094820056eb838f58a235763b63df3f5c41" ;;
        linux-arm64)  echo "46d98eefc3bb09b5559140b8c65a128742d6ec0bddc3d4f8851b8eee5de9b660" ;;
        linux-s390x)  echo "6ab0e13093d6dfbf5234cd935c24615de4658b17afc787542e62f8eaf5e3ccc7" ;;
        darwin-amd64) echo "215402eabbdde708982724e189b77d53bec5cd0d1ff767b310268b0bc6375269" ;;
        darwin-arm64) echo "ace3d1135dbb29283eb04c224514276e7627e05b577606b6fc0b8a9e610bf4d6" ;;
        *)
            echo "ERROR: no bazeldnf binary available for platform $1" >&2
            return 1
            ;;
    esac
}

function bazeldnf::detect_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$(uname -m)" in
        x86_64)  arch="amd64" ;;
        aarch64) arch="arm64" ;;
        arm64)   arch="arm64" ;;
        s390x)   arch="s390x" ;;
        *)
            echo "ERROR: unsupported architecture $(uname -m)" >&2
            return 1
            ;;
    esac
    echo "${os}-${arch}"
}

function bazeldnf::checksum() {
    if command -v sha256sum &>/dev/null; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

function bazeldnf::install() {
    local platform
    platform="$(bazeldnf::detect_platform)"

    local expected_sha
    expected_sha="$(bazeldnf::sha256_for_platform "${platform}")"

    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local repo_root="${script_dir}/.."
    local tools_dir="${repo_root}/_out/tools"
    local binary="${tools_dir}/bazeldnf"

    if [ -x "${binary}" ]; then
        local actual_sha
        actual_sha="$(bazeldnf::checksum "${binary}")"
        if [ "${actual_sha}" = "${expected_sha}" ]; then
            export PATH="${tools_dir}:${PATH}"
            return 0
        fi
        echo "Cached bazeldnf checksum mismatch, re-downloading..."
    fi

    local artifact="bazeldnf-${BAZELDNF_VERSION}-${platform}"
    local url="https://github.com/${BAZELDNF_REPO}/releases/download/${BAZELDNF_VERSION}/${artifact}"

    echo "Downloading bazeldnf ${BAZELDNF_VERSION} for ${platform}..."
    mkdir -p "${tools_dir}"
    curl -sSL -o "${binary}" "${url}"
    chmod +x "${binary}"

    local actual_sha
    actual_sha="$(bazeldnf::checksum "${binary}")"

    if [ "${actual_sha}" != "${expected_sha}" ]; then
        echo "ERROR: bazeldnf checksum verification failed" >&2
        echo "  expected: ${expected_sha}" >&2
        echo "  actual:   ${actual_sha}" >&2
        rm -f "${binary}"
        return 1
    fi

    echo "bazeldnf ${BAZELDNF_VERSION} installed to ${binary}"
    export PATH="${tools_dir}:${PATH}"
}

bazeldnf::install
