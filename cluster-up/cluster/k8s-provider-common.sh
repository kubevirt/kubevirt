#!/usr/bin/env bash

set -e

# shellcheck source=cluster-up/cluster/ephemeral-provider-common.sh
source "${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh"



function deploy_kwok() {
    if [[ ${KUBEVIRT_DEPLOY_KWOK} == "true" ]]; then
        $kubectl create -f /opt/kwok/kwok.yaml
        $kubectl create -f /opt/kwok/stage-fast.yaml
    fi
}

# copy_istio_cni_conf_files copy the generated Istio CNI net conf file
# (at '/etc/cni/multus/net.d/') to where Multus expect CNI net conf files ('/etc/cni/net.d/')
function copy_istio_cni_conf_files() {
    if [ "$KUBEVIRT_DEPLOY_ISTIO" == "true" ] && [ "$KUBEVIRT_WITH_CNAO" == "true" ]; then
        for nodeNum in $(seq -f "%02g" 1 $KUBEVIRT_NUM_NODES); do
            $ssh node${nodeNum} -- "until ls /etc/cni/multus > /dev/null 2>&1; do sleep 1; done"
            $ssh node${nodeNum} -- sudo cp -uv /etc/cni/multus/net.d/*istio*.conf /etc/cni/net.d/
        done
    fi
}

# configure Prometheus to select kubevirt prometheusrules
function configure_prometheus() {
    if [[ $KUBEVIRT_DEPLOY_PROMETHEUS == "true" ]] && $kubectl get crd prometheuses.monitoring.coreos.com; then
        _kubectl patch prometheus k8s -n monitoring --type='json' -p='[{"op": "replace", "path": "/spec/ruleSelector", "value":{}}, {"op": "replace", "path": "/spec/ruleNamespaceSelector", "value":{"matchLabels": {}}}]'
    fi
}


function wait_for_kwok_ready() {
    if [ "KUBEVIRT_DEPLOY_KWOK" == "true" ]; then
        $kubectl wait deployment -n kube-system kwok-controller --for condition=Available --timeout=200s
    fi
}

function configure_cpu_manager() {
    if [ ${KUBEVIRT_CPU_MANAGER_POLICY} == "static" ]; then
        for node in $($kubectl get nodes -l "node-role.kubernetes.io/worker" --no-headers -o custom-columns=":metadata.name" | tr -d '\r'); do
            # FIXME Replace with kubelet config drop ins once all providers are using k8s >= 1.28
            # https://kubernetes.io/docs/tasks/administer-cluster/kubelet-config-file/#kubelet-conf-d
            $kubectl drain ${node}
            $ssh ${node} -- sudo systemctl stop kubelet
            # FIXME ${ssh} is broken when using HereDocs, fix and replace this mess if possible.
            # https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/#configuration
            $ssh ${node} -- "sudo rm -f /var/lib/kubelet/cpu_manager_state && sudo echo -e 'cpuManagerPolicy: static\nkubeReserved:\n  cpu: \"1\"\n memory: \"1Gi\"\ncpuManagerPolicyOptions:\n  full-pcpus-only: \"true\"' | sudo tee -a /var/lib/kubelet/config.yaml && sudo sed -i 's/cpuManagerReconcilePeriod\:\ 0s/cpuManagerReconcilePeriod\:\ 5s/g' /var/lib/kubelet/config.yaml"
            $ssh ${node} -- sudo systemctl start kubelet
            $kubectl label --overwrite node/${node} cpumanager=true
            $kubectl uncordon ${node}
        done
    fi
}

function up() {
    params=$(_add_common_params)
    if echo "$params" | grep -q ERROR; then
        echo -e "$params"
        exit 1
    fi
    eval ${_cli:?} run $params

    ${_cli} scp --prefix $provider_prefix /etc/kubernetes/admin.conf - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    kubectl config set-cluster kubernetes --server="https://$(_main_ip):$(_port k8s)"
    kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

    # Workaround https://github.com/containers/conmon/issues/315 by not dumping the file to stdout for the time being
    if [[ ${_cri_bin} = podman* ]]; then
        k8s_version=$(kubectl get node node01 --no-headers -o=custom-columns=VERSION:.status.nodeInfo.kubeletVersion)
        curl -Ls "https://dl.k8s.io/release/${k8s_version}/bin/linux/amd64/kubectl" -o ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    else
        ${_cli} scp --prefix ${provider_prefix:?} /usr/bin/kubectl - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    fi

    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl

    # Make sure that local config is correct
    prepare_config
    ssh="${_cli} --prefix $provider_prefix ssh"
    kubectl="$ssh node01 -- sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf"

    # For multinode cluster Label all the non control-plane nodes as workers,
    # for one node cluster label control-plane with 'control-plane,worker' roles
    if [ "$KUBEVIRT_NUM_NODES" -gt 1 ]; then
        label="!node-role.kubernetes.io/control-plane"
    else
        label="node-role.kubernetes.io/control-plane"
    fi
    $kubectl label node -l $label node-role.kubernetes.io/worker=''

    configure_prometheus
    configure_cpu_manager

    deploy_kwok

    until wait_for_kwok_ready; do
        echo "Waiting for cluster components..."
        sleep 5
    done

    # FIXME: remove 'copy_istio_cni_conf_files()' as soon as [1] and [2] are resolved
    # [1] https://github.com/kubevirt/kubevirtci/issues/906
    # [2] https://github.com/k8snetworkplumbingwg/multus-cni/issues/982
    copy_istio_cni_conf_files
}
