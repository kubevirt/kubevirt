#!/usr/bin/env bash

set -euo pipefail

source hack/common.sh

if [ "${JOB_TYPE}" == "stdci" ]; then
    KUBECONFIG=${KUBEVIRTCI_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
fi

./${TEST_OUT_PATH}/func-tests.test -ginkgo.v -test.timeout 120m -kubeconfig="${KUBECONFIG}"
