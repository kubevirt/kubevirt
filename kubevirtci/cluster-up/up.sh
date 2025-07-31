#!/usr/bin/env bash

function validate_single_stack_ipv6() {
    local kube_ns="kube-system"
    local pod_label="calico-kube-controllers"

    echo "validating provider is single stack IPv6"
    until _kubectl wait --for=condition=Ready pod --timeout=10s -n $kube_ns -lk8s-app=${pod_label}; do sleep 1; done > /dev/null 2>&1

    local pod=$(_kubectl get pods -n ${kube_ns} -lk8s-app=${pod_label} -o=custom-columns=NAME:.metadata.name --no-headers)
    local primary_ip=$(_kubectl get pod -n ${kube_ns} ${pod} -ojsonpath="{ @.status.podIP }")

    if [[ ! ${primary_ip} =~ fd00 ]]; then
        echo "error: single stack primary ip ($primary_ip) is not IPv6 as expected"
        exit 1
    fi

    if _kubectl get pod -n ${kube_ns} ${pod} -ojsonpath="{ @.status.podIPs[1] }" > /dev/null 2>&1; then
        echo "error: single stack cluster expected, podIPs"
        _kubectl get pod -n ${kube_ns} ${pod} -ojsonpath="{ @.status.podIPs }"
        exit 1
    fi
}

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/"
        echo "$(pwd)/"
    )"
fi


source ${KUBEVIRTCI_PATH}hack/common.sh
source ${KUBEVIRTCI_CLUSTER_PATH}/$KUBEVIRT_PROVIDER/provider.sh
up

if [ ${KUBEVIRT_SINGLE_STACK} == true ]; then
    validate_single_stack_ipv6
fi
