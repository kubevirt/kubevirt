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
# Copyright the KubeVirt Authors.
#
#

function update_github_source_tarball_signature() {
    local src_tarball_file
    src_tarball_file="${DOCKER_TAG}.tar.gz"
    src_tarball_signature_file="${src_tarball_file}.asc"

    if ! gh release download "$DOCKER_TAG" --repo "$GITHUB_REPOSITORY" --pattern="${src_tarball_signature_file}" --clobber --output "/tmp/${src_tarball_signature_file}"; then
        upload_github_source_tarball_signature
    else
        gh release download "$DOCKER_TAG" --repo "$GITHUB_REPOSITORY" --archive=tar.gz --clobber --output "/tmp/${src_tarball_file}"
        if ! gpg --verify --local-user "${GPG_USER_ID}" "/tmp/${src_tarball_signature_file}" "/tmp/${src_tarball_file}"; then
            upload_github_source_tarball_signature
        fi
    fi

}

function upload_github_source_tarball_signature() {
    local src_tarball_file
    src_tarball_file="${DOCKER_TAG}.tar.gz"

    # 1. download source tarball
    # example download url: https://github.com/kubevirt/kubevirt/archive/refs/tags/v1.3.0-beta.0.tar.gz
    gh release download "$DOCKER_TAG" --repo "$GITHUB_REPOSITORY" --archive=tar.gz --clobber --output "/tmp/${src_tarball_file}"

    # 2. sign with private key (to verify the signature a prerequisite is that the public key is uploaded)

    [ -f "/tmp/${src_tarball_file}.asc" ] && rm "/tmp/${src_tarball_file}.asc"
    gpg --armor --detach-sign --local-user "${GPG_USER_ID}" --output "/tmp/${src_tarball_file}.asc" "/tmp/${src_tarball_file}"

    # 3. upload the detached signature
    gh release upload --repo "$GITHUB_REPOSITORY" --clobber "$DOCKER_TAG" "/tmp/${src_tarball_file}.asc"
}
