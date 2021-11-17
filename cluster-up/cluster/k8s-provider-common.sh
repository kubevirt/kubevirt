#!/usr/bin/env bash

set -e

# shellcheck source=cluster-up/cluster/ephemeral-provider-common.sh
source "${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh"

function deploy_cnao() {
    if [ "$KUBEVIRT_WITH_CNAO" == "true" ] || [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" == "true" ]; then
        $kubectl create -f /opt/cnao/namespace.yaml
        $kubectl create -f /opt/cnao/network-addons-config.crd.yaml
        $kubectl create -f /opt/cnao/operator.yaml

        if [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" != "true" ]; then
            $kubectl create -f /opt/cnao/network-addons-config-example.cr.yaml
        fi

        # Install whereabouts on CNAO lanes
        $kubectl create -f /opt/whereabouts
    fi
}

function wait_for_cnao_ready() {
    if [ "$KUBEVIRT_WITH_CNAO" == "true" ] || [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" == "true" ]; then
        $kubectl wait deployment -n cluster-network-addons cluster-network-addons-operator --for condition=Available --timeout=200s
        if [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" != "true" ]; then
            $kubectl wait networkaddonsconfig cluster --for condition=Available --timeout=200s
        fi
    fi
}

function deploy_istio() {
    if [ "$KUBEVIRT_DEPLOY_ISTIO" == "true" ] && [[ $KUBEVIRT_PROVIDER =~ k8s-1\.1.* ]]; then
        echo "ERROR: Istio is not supported on kubevirtci version < 1.20"
        exit 1

    elif [ "$KUBEVIRT_DEPLOY_ISTIO" == "true" ]; then
        if [ "$KUBEVIRT_WITH_CNAO" == "true" ]; then
            $kubectl create -f /opt/istio/istio-operator-with-cnao.cr.yaml
        else
            $kubectl create -f /opt/istio/istio-operator.cr.yaml
        fi
    fi
}

function wait_for_istio_ready() {
    if [ "$KUBEVIRT_DEPLOY_ISTIO" == "true" ]; then
        istio_operator_ns=istio-system
        retries=0
        max_retries=20
        while [[ $retries -lt $max_retries ]]; do
            echo "waiting for istio-operator to be healthy"
            sleep 5
            health=$($kubectl -n $istio_operator_ns get istiooperator istio-operator -o jsonpath="{.status.status}")
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

function deploy_cdi() {
    if [ "$KUBEVIRT_DEPLOY_CDI" == "true" ]; then
        $kubectl create -f /opt/cdi-*-operator.yaml
        $kubectl create -f /opt/cdi-*-cr.yaml
    fi
}

function wait_for_cdi_ready() {
    if [ "$KUBEVIRT_DEPLOY_CDI" == "true" ]; then
        while [ "$($kubectl get pods --namespace cdi | grep -c 'cdi-')" -lt 4 ]; do
            $kubectl get pods --namespace cdi
            sleep 10
        done
        $kubectl wait --for=condition=Ready pod --timeout=180s --all --namespace cdi
    fi
}

function up() {
    params=$(_add_common_params)
    if echo "$params" | grep -q ERROR; then
        echo -e "$params"
        exit 1
    fi
    eval ${_cli:?} run $params

    # Copy k8s config and kubectl
    ${_cli} scp --prefix ${provider_prefix:?} /usr/bin/kubectl - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    ${_cli} scp --prefix $provider_prefix /etc/kubernetes/admin.conf - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster kubernetes --server="https://$(_main_ip):$(_port k8s)"
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

    deploy_cnao
    deploy_istio
    deploy_cdi

    until wait_for_cnao_ready && wait_for_istio_ready && wait_for_cdi_ready; do
        echo "Waiting for cluster components..."
        sleep 5
    done

}
