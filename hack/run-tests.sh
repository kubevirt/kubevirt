#!/usr/bin/env bash

set -euo pipefail

source hack/common.sh

CSV_FILE=

if [ "${JOB_TYPE}" == "stdci" ]; then
    KUBECONFIG=${KUBEVIRTCI_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    source ./hack/upgrade-stdci-config
    
    # check if CSV test is requested (if this is run right after upgrade-test.sh)
    CSV_FILE=./test-out/clusterserviceversion.yaml 
    if [ -f ${CSV_FILE} ]; then
        echo "** enable CSV test **"
        export TEST_KUBECTL_CMD="${CMD}"
        export TEST_CSV_FILE="${CSV_FILE}"
    fi
fi

./${TEST_OUT_PATH}/func-tests.test -ginkgo.v -test.timeout 120m -kubeconfig="${KUBECONFIG}" 

if [ -f ${CSV_FILE} ]; then
  rm -f ${CSV_FILE}
fi  
