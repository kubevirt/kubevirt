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
# Copyright The KubeVirt Authors.
#

set -e

source hack/common.sh
source hack/config.sh

PLATFORM=$(uname -m)
case ${PLATFORM} in
x86_64* | i?86_64* | amd64*)
    ARCH="amd64"
    ;;
aarch64* | arm64*)
    ARCH="arm64"
    ;;
s390x)
    ARCH="s390x"
    ;;
*)
    echo "invalid Arch, only support x86_64, aarch64 and s390x"
    exit 1
    ;;
esac

function build_func_tests() {
    echo "building functional tests"
    rm -rf "${TESTS_OUT_DIR}"
    mkdir -p "${TESTS_OUT_DIR}/"
    GOOS=linux GOARCH=${ARCH} go_build -tags "${KUBEVIRT_GO_BUILD_TAGS}" -o "${TESTS_OUT_DIR}/ginkgo" vendor/github.com/onsi/ginkgo/v2/ginkgo/main.go
    GOOS=linux GOARCH=${ARCH} GOPROXY=off go test -tags "${KUBEVIRT_GO_BUILD_TAGS}" -c -o "${TESTS_OUT_DIR}/tests.test" "${KUBEVIRT_DIR}/tests"
    GOOS=linux GOARCH=${ARCH} go_build -tags "${KUBEVIRT_GO_BUILD_TAGS}" -o "${TESTS_OUT_DIR}/junit-merger" tools/junit-merger/junit-merger.go
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    build_func_tests
fi
