#!/usr/bin/env bash

set -e

source ${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh

function up() {
    params=$(echo $(_add_common_params))
    if [[ ! -z $(echo $params | grep ERROR) ]]; then
        echo -e $params
        exit 1
    fi
    eval ${_cli} run $params

    # Copy k8s config and kubectl
    ${_cli} scp --prefix $provider_prefix /usr/bin/kubectl - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    ${_cli} scp --prefix $provider_prefix /etc/kubernetes/admin.conf - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster kubernetes --server=https://$(_main_ip):$(_port k8s)
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config


    kubectl="${_cli} --prefix $provider_prefix ssh node01 -- sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf"

    # For multinode cluster Label all the non master nodes as workers,
    # for one node cluster label master with 'master,worker' roles
    if [ "$KUBEVIRT_NUM_NODES" -gt 1 ]; then
        label="!node-role.kubernetes.io/master"
    else
        label="node-role.kubernetes.io/master"
    fi
    $kubectl label node -l $label node-role.kubernetes.io/worker=''

    # Activate cluster-network-addons-operator if flag is passed
    if [ "$KUBEVIRT_WITH_CNAO" == "true" ] || [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" == "true" ]; then

        $kubectl create -f /opt/cnao/namespace.yaml
        $kubectl create -f /opt/cnao/network-addons-config.crd.yaml
        $kubectl create -f /opt/cnao/operator.yaml
        $kubectl wait deployment -n cluster-network-addons cluster-network-addons-operator --for condition=Available --timeout=200s

        if [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" != "true" ]; then

            $kubectl create -f /opt/cnao/network-addons-config-example.cr.yaml
            $kubectl wait networkaddonsconfig cluster --for condition=Available --timeout=200s
        fi
    fi

    if [ "$KUBEVIRT_DEPLOY_ISTIO" == "true" ] && [[ $KUBEVIRT_PROVIDER =~ k8s-1\.1.* ]]; then
        echo "ERROR: Istio is not supported on kubevirtci version < 1.20"
        exit 1

    elif [ "$KUBEVIRT_DEPLOY_ISTIO" == "true" ]; then
        if [ "$KUBEVIRT_WITH_CNAO" == "true" ]; then
            $kubectl create -f /opt/istio/istio-operator-with-cnao.cr.yaml
        else
            $kubectl create -f /opt/istio/istio-operator.cr.yaml
        fi
        
        istio_operator_ns=istio-system
        retries=0
        max_retries=20
        while [[ $retries -lt $max_retries ]]; do
            echo "waiting for istio-operator to be healthy"
            sleep 5
            health=$(kubectl -n $istio_operator_ns get istiooperator istio-operator -o jsonpath="{.status.status}")
            if [[ $health == "HEALTHY" ]]; then
                break
            fi
            retries=$((retries + 1))
        done
        if [ $retries == $max_retries ]; then
            echo "waiting istio-operator to be healthy failed"
            exit 1
        fi
    fi
}
