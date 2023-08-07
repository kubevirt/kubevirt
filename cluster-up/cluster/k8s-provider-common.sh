#!/usr/bin/env bash

set -e

# shellcheck source=cluster-up/cluster/ephemeral-provider-common.sh
source "${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh"


#if UNLIMITEDSWAP is set to true - Kubernetes workloads can use as much swap memory as they request, up to the system limit.
#otherwise Kubernetes workloads can use as much swap memory as they request, up to the system limit by default
function configure_swap_memory () {
    if [ "$KUBEVIRT_SWAP_ON" == "true" ] ;then
      for nodeNum in $(seq -f "%02g" 1 $KUBEVIRT_NUM_NODES); do
          if [ ! -z $KUBEVIRT_SWAP_SIZE_IN_GB  ]; then
            $ssh node${nodeNum} -- sudo dd if=/dev/zero of=/swapfile count=$KUBEVIRT_SWAP_SIZE_IN_GB bs=1G
            $ssh node${nodeNum} -- sudo mkswap /swapfile
          fi

          $ssh node${nodeNum} -- sudo swapon -a

          if [ ! -z $KUBEVIRT_SWAPPINESS ]; then
            $ssh node${nodeNum} -- "sudo /bin/su -c \"echo vm.swappiness = $KUBEVIRT_SWAPPINESS >> /etc/sysctl.conf\""
            $ssh node${nodeNum} -- sudo sysctl vm.swappiness=$KUBEVIRT_SWAPPINESS
          fi

          if [ $KUBEVIRT_UNLIMITEDSWAP == "true" ]; then
            $ssh node${nodeNum} -- "sudo sed -i ':a;N;\$!ba;s/memorySwap: {}/memorySwap:\n  swapBehavior: UnlimitedSwap/g'  /var/lib/kubelet/config.yaml"
            $ssh node${nodeNum} -- sudo systemctl restart kubelet
          fi
      done
    fi
}

function configure_ksm_module () {
    if [ "$KUBEVIRT_KSM_ON" == "true" ] ;then
      for nodeNum in $(seq -f "%02g" 1 $KUBEVIRT_NUM_NODES); do
        $ssh node${nodeNum} -- "echo 1 | sudo tee /sys/kernel/mm/ksm/run >/dev/null"
          if [ ! -z $KUBEVIRT_KSM_SLEEP_BETWEEN_SCANS_MS ]; then
            $ssh node${nodeNum} -- "echo ${KUBEVIRT_KSM_SLEEP_BETWEEN_SCANS_MS} | sudo tee /sys/kernel/mm/ksm/sleep_millisecs >/dev/null "
          fi
          if [ ! -z $KUBEVIRT_KSM_PAGES_TO_SCAN ]; then
            $ssh node${nodeNum} -- "echo ${KUBEVIRT_KSM_PAGES_TO_SCAN} | sudo tee /sys/kernel/mm/ksm/pages_to_scan >/dev/null "
          fi
      done
    fi
}

function configure_memory_overcommitment_behavior () {
  configure_swap_memory
  configure_ksm_module
}

function deploy_cnao() {
    if [ "$KUBEVIRT_WITH_CNAO" == "true" ] || [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" == "true" ]; then
        $kubectl create -f /opt/cnao/namespace.yaml
        $kubectl create -f /opt/cnao/network-addons-config.crd.yaml
        $kubectl create -f /opt/cnao/operator.yaml

        if [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" != "true" ]; then
            create_network_addons_config
        fi

        # Install whereabouts on CNAO lanes
        $kubectl create -f /opt/whereabouts
    fi
}

function create_network_addons_config() {
    local nac="/opt/cnao/network-addons-config-example.cr.yaml"
    if [ "$KUBEVIRT_WITH_MULTUS_V3" == "true" ]; then
        local no_multus_nac="/opt/cnao/no-multus-nac.yaml"
        $ssh node01 -- "awk '!/multus/' ${nac} | sudo tee ${no_multus_nac}"
        nac=${no_multus_nac}
    fi

     $kubectl apply -f ${nac}
}

function wait_for_cnao_ready() {
    if [ "$KUBEVIRT_WITH_CNAO" == "true" ] || [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" == "true" ]; then
        $kubectl wait deployment -n cluster-network-addons cluster-network-addons-operator --for condition=Available --timeout=200s
        if [ "$KUBVIRT_WITH_CNAO_SKIP_CONFIG" != "true" ]; then
            $kubectl wait networkaddonsconfig cluster --for condition=Available --timeout=200s
        fi
    fi
}

function deploy_multus() {
    if [ "$KUBEVIRT_WITH_MULTUS_V3" == "true" ]; then
        $kubectl create -f /opt/multus/multus.yaml
    fi
}

function wait_for_multus_ready() {
    if [ "$KUBEVIRT_WITH_MULTUS_V3" == "true" ]; then
        $kubectl rollout status -n kube-system ds/kube-multus-ds --timeout=200s
    fi
}

function deploy_istio() {
    if [ "$KUBEVIRT_DEPLOY_ISTIO" == "true" ]; then
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

# copy_istio_cni_conf_files copy the generated Istio CNI net conf file
# (at '/etc/cni/multus/net.d/') to where Multus expect CNI net conf files ('/etc/cni/net.d/')
function copy_istio_cni_conf_files() {
    if [ "$KUBEVIRT_DEPLOY_ISTIO" == "true" ] && [ "$KUBEVIRT_WITH_CNAO" == "true" ]; then
        for nodeNum in $(seq -f "%02g" 1 $KUBEVIRT_NUM_NODES); do
            $ssh node${nodeNum} -- sudo cp -uv /etc/cni/multus/net.d/*istio*.conf /etc/cni/net.d/
        done
    fi
}

function deploy_cdi() {
    if [ "$KUBEVIRT_DEPLOY_CDI" == "true" ]; then
        if [ -n "${KUBEVIRT_CUSTOM_CDI_VERSION}" ]; then
            $ssh node01 -- 'sudo sed --regexp-extended -i s/v[0-9]+\.[0-9]+\.[0-9]+\(.*\)?$/'"$KUBEVIRT_CUSTOM_CDI_VERSION"'/g /opt/cdi-*-operator.yaml'
        fi

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

    configure_memory_overcommitment_behavior

    deploy_cnao
    deploy_multus
    deploy_istio
    deploy_cdi

    until wait_for_cnao_ready && wait_for_istio_ready && wait_for_cdi_ready && wait_for_multus_ready; do
        echo "Waiting for cluster components..."
        sleep 5
    done

    # FIXME: remove 'copy_istio_cni_conf_files()' as soon as [1] and [2] are resolved
    # [1] https://github.com/kubevirt/kubevirtci/issues/906
    # [2] https://github.com/k8snetworkplumbingwg/multus-cni/issues/982
    copy_istio_cni_conf_files
}
