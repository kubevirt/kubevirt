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
            $ssh node${nodeNum} -- "until ls /etc/cni/multus/net.d/*istio*.conf > /dev/null 2>&1; do sleep 1; done"
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

function configure_nfs() {
    if [[ "$KUBEVIRT_DEPLOY_NFS_CSI" == "true" ]] && [[ -n "$KUBEVIRT_NFS_DIR" ]]; then
        ${_cri_bin} run --privileged --rm -v /:/hostroot \
            --entrypoint /bin/sh ${_cli_container} \
            -c "mkdir -p /hostroot/${KUBEVIRT_NFS_DIR} && chmod 777 /hostroot/${KUBEVIRT_NFS_DIR}"

        ${_cri_bin} run --privileged --rm -v $KUBEVIRT_NFS_DIR:/nfsdir \
            --entrypoint /bin/sh ${_cli_container} \
            -c 'for disk in disk1 disk2 disk3 disk4 disk5 disk6 disk7 disk8 disk9 disk10 extraDisk1 extraDisk2; do \
            mkdir -p /nfsdir/${disk} && chmod 777 /nfsdir/${disk}; done'
    fi
}


function add_image_volume_feature_gate () {
  if [[  ($KUBEVIRT_PROVIDER =~ k8s-1\.1.*) ||  ($KUBEVIRT_PROVIDER =~ k8s-1.29) ||  ($KUBEVIRT_PROVIDER =~ k8s-1.30) ]]; then
      echo "ImageVolume feature is supported only on Kubernetes version >= 1.31"
      return
  fi

    for nodeNum in $(seq -f "%02g" 1 $KUBEVIRT_NUM_NODES); do
        if ! $ssh node${nodeNum} -- grep -q "feature-gates:" /var/lib/kubelet/config.yaml; then
          echo "feature-gates section not found, adding it"
          $ssh node${nodeNum} -- "sudo /bin/su -c \"echo -e 'featureGates:' >> /var/lib/kubelet/config.yaml\""
        fi

        if ! $ssh node${nodeNum} -- grep -q "  ImageVolume=true" /var/lib/kubelet/config.yaml; then
          echo "Adding ImageVolume=true under feature-gates"
          $ssh node${nodeNum} -- "sudo sed -i ':a;N;\$!ba;s/featureGates:/featureGates:\n\ \ ImageVolume: true/g' /var/lib/kubelet/config.yaml"
        fi

        $ssh node${nodeNum} -- sudo systemctl restart kubelet

        # Check for the existence of the kube-apiserver manifest and modify it
        if $ssh node${nodeNum} -- test -f /etc/kubernetes/manifests/kube-apiserver.yaml; then
          echo "Found kube-apiserver.yaml on node${nodeNum}, checking for feature-gates"

          if ! $ssh node${nodeNum} -- grep -q " --feature-gates=ImageVolume=true" /etc/kubernetes/manifests/kube-apiserver.yaml; then
            echo "Adding --feature-gates=ImageVolume=true to kube-apiserver.yaml on node${nodeNum}"
            $ssh node${nodeNum} -- "sudo sed -i ':a;N;\$!ba;s/- kube-apiserver/- kube-apiserver\n\ \ \ \ - --feature-gates=ImageVolume=true/g' /etc/kubernetes/manifests/kube-apiserver.yaml"
            echo "Waiting for the API server to be ready..."
            until kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[0].spec.containers[0].command}' | grep -q -- '--feature-gates=ImageVolume=true'; do
              echo "API server is not ready yet or flag is not set, waiting..."
              sleep 5
            done
            echo "API server is ready and feature-gates flag is set."
            echo "API server is back online."
          else
            echo "--feature-gates=ImageVolume=true already present in kube-apiserver.yaml on node${nodeNum}"
          fi
        else
          echo "kube-apiserver.yaml not found on node${nodeNum}, skipping modification"
        fi
    done
}

function up() {
    params=$(_add_common_params)
    if echo "$params" | grep -q ERROR; then
        echo -e "$params"
        exit 1
    fi
    eval ${_cli:?} run $params

    cli_scp_command "${_cli} scp --prefix $provider_prefix /etc/kubernetes/admin.conf" ".kubeconfig"
    cli_scp_command "${_cli} scp --prefix ${provider_prefix:?} /usr/bin/kubectl" ".kubectl"
    change_permissions

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    _kubectl config set-cluster kubernetes --server="https://$(_main_ip):$(_port k8s)"
    _kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

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

    add_image_volume_feature_gate
    configure_prometheus

    deploy_kwok

    until wait_for_kwok_ready; do
        echo "Waiting for cluster components..."
        sleep 5
    done

    # FIXME: remove 'copy_istio_cni_conf_files()' as soon as [1] and [2] are resolved
    # [1] https://github.com/kubevirt/kubevirtci/issues/906
    # [2] https://github.com/k8snetworkplumbingwg/multus-cni/issues/982
    copy_istio_cni_conf_files

    configure_nfs
}

# The scp command for docker and podman is different, in order to avoid segmentation fault
function cli_scp_command() {
    prefix_cli="$1"
    destination_file="$2"
    # Workaround https://github.com/containers/conmon/issues/315 by not dumping file content to stdout
    if [[ ${_cri_bin} = podman* ]]; then
        $prefix_cli /kubevirtci_config/$destination_file
    else
        $prefix_cli - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/$destination_file
    fi
}

function change_permissions() {
    if [[ ${_cri_bin} = podman* ]]; then
        args="-v ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER:/kubevirtci_config"
        ${_cri_bin} run --privileged --rm $args \
            --entrypoint /bin/sh ${_cli_container} \
            -c "chmod 755 /kubevirtci_config/.kubectl"
        ${_cri_bin} run --privileged --rm $args \
            --entrypoint /bin/sh ${_cli_container} \
            -c "chmod 766 /kubevirtci_config/.kubeconfig"
    fi
}
