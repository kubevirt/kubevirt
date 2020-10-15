#!/bin/bash

set -euo pipefail

export PATH=$PATH:$HOME/gopath/bin
JOB_TYPE="${JOB_TYPE:-}"

if [ "${JOB_TYPE}" == "travis" ]; then
    go get -v -t ./...
    go get github.com/mattn/goveralls
    go get -v github.com/onsi/ginkgo/ginkgo
    go get -v github.com/onsi/gomega
    go get -u github.com/evanphx/json-patch
    go mod vendor
    PACKAGE_PATH="pkg/"
    ginkgo -r -cover ${PACKAGE_PATH}
else
    test_path="tests/func-tests"
    (cd $test_path; GOFLAGS= go get github.com/onsi/ginkgo/ginkgo)
    (cd $test_path; GOFLAGS= go get github.com/onsi/gomega)
    (cd $test_path; go mod  tidy; go mod vendor)
    test_out_path=${test_path}/_out
    mkdir -p ${test_out_path}
    (cd $test_path; ginkgo build .)
    mv ${test_path}/func-tests.test ${test_out_path}
fi
