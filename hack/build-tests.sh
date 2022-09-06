#!/bin/bash

set -euxo pipefail

export PATH=$PATH:$HOME/gopath/bin
JOB_TYPE="${JOB_TYPE:-}"

if [ "${JOB_TYPE}" == "travis" ]; then
    go get -v -t ./...
    go install github.com/mattn/goveralls@latest
    go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.3
    go mod vendor
    PKG_PACKAGE_PATH="./pkg/"
    CONTROLLERS_PACKAGE_PATH="./controllers/"
    mkdir -p coverprofiles
    # Workaround - run tests on webhooks first to prevent failure when running all the test in the following line.
    ginkgo -r ${PKG_PACKAGE_PATH}webhooks
    ginkgo -cover -output-dir=./coverprofiles -coverprofile=cover.coverprofile -r ${PKG_PACKAGE_PATH} -r ${CONTROLLERS_PACKAGE_PATH}
else
    test_path="tests/func-tests"
    (cd $test_path; GOFLAGS='' go install github.com/onsi/ginkgo/v2/ginkgo@latest)
    (cd $test_path; go mod tidy; go mod vendor)
    test_out_path=${test_path}/_out
    mkdir -p ${test_out_path}
    (cd $test_path; ginkgo build .)
    mv ${test_path}/func-tests.test ${test_out_path}
fi
