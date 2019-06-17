#!/bin/bash

source hack/common.sh

VIRT_POD=`${CMD} get pods -n kubevirt | grep virt-operator | head -1 | awk '{ print $1 }'`
CDI_POD=`${CMD} get pods -n cdi | grep cdi-operator | head -1 | awk '{ print $1 }'`
NETWORK_ADDONS_POD=`${CMD} get pods -n cluster-network-addons-operator | grep cluster-network-addons-operator | head -1 | awk '{ print $1 }'`
HCO_POD=`${CMD} get pods -n kubevirt-hyperconverged | grep hyperconverged-cluster-operator | head -1 | awk '{ print $1 }'`
SSP_POD=`${CMD} get pods -n kubevirt-hyperconverged | grep kubevirt-ssp-operator | head -1 | awk '{ print $1 }'`

function new_test() {
    name=$1

    printf "%0.s=" {1..80}
    echo
    echo ${name}
}

function assert_condition() {
    podname=$1
    condition=$2
    namespace=$3
    timeout=$4

    echo "Wait until ${podname} reports ${condition} condition"
    if ${CMD} wait pod ${podname} --for condition=${condition} -n ${namespace} --timeout=${timeout}; then
        echo 'OK'
    else
        echo "${podname} status has not reached ${condition} condition within the timeout. Actual state:"
        ${CMD} logs ${podname} -n ${namespace}
        echo 'FAILED'
        exit 1
    fi
}

new_test 'Test operators'
assert_condition $HCO_POD Ready kubevirt-hyperconverged 60s
assert_condition $VIRT_POD Ready kubevirt 60s
assert_condition $CDI_POD Ready cdi 60s
assert_condition $NETWORK_ADDONS_POD Ready cluster-network-addons-operator 60s
assert_condition $SSP_POD Ready kubevirt-hyperconverged 60s
