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


set -exo pipefail

GITHUB_FQDN=github.com
API_REF_REPO=${API_REF_REPO:-kubevirt/api-reference}
API_REF_DIR=/tmp/api-reference
GITHUB_IO_FQDN="https://kubevirt.github.io/api-reference"

TARGET_DIR="$PULL_BASE_REF"
if [ -n "${DOCKER_TAG}" ]; then
    TARGET_DIR="$DOCKER_TAG"
fi

git clone \
    "https://${GIT_USER_NAME}@${GITHUB_FQDN}/${API_REF_REPO}.git" \
    "${API_REF_DIR}" >/dev/null 2>&1
rm -rf "${API_REF_DIR}/${TARGET_DIR:?}/"*
mkdir -p ${API_REF_DIR}/${TARGET_DIR}
cp -f _out/apidocs/html/*.html "${API_REF_DIR}/${TARGET_DIR}/"

cd "${API_REF_DIR}"

# Generate README.md file
cat >README.md <<__EOF__
# KubeVirt API Reference

Content of this repository is generated from OpenAPI specification of
[KubeVirt project](https://github.com/kubevirt/kubevirt) .

## KubeVirt API References

* [main](${GITHUB_IO_FQDN}/main/index.html)
__EOF__
find * -type d -regex "^v[0-9.]*" \
    -exec echo "* [{}](${GITHUB_IO_FQDN}/{}/index.html)" \; | sort -r --version-sort -t '[' --key 2 >>README.md

git config user.email "${GIT_AUTHOR_EMAIL:-kubevirtbot@redhat.com}"
git config user.name "${GIT_AUTHOR_NAME:-kubevirt-bot}"

# NOTE: exclude index.html from match, because it is static except commit hash.
if git status --porcelain | grep -v "index[.]html" | grep --quiet "^ [AM]"; then
    git add -A README.md "${TARGET_DIR}"/*.html
    git commit --message "API Reference update by KubeVirt Prow build ${BUILD_ID}"

    git push origin master >/dev/null 2>&1
    echo "API Reference updated for ${TARGET_DIR}."
else
    echo "API Reference hasn't changed."
fi
