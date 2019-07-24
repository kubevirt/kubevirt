#!/bin/bash -e

CONTROL_PLANE_CMD="docker exec -it -d ${CLUSTER_NAME}-control-plane"
MANIFESTS_DIR="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/manifests"

function wait_containers_ready {
    echo "Waiting for all containers to become ready ..."
    kubectl wait --for=condition=Ready pod --all -n kube-system --timeout 12m
}

#move the pf to the node
mkdir -p /var/run/netns/
export pid="$(docker inspect -f '{{.State.Pid}}' ${CLUSTER_NAME}-control-plane)"
ln -sf /proc/$pid/ns/net "/var/run/netns/${CLUSTER_NAME}-control-plane"

sriov_pfs=( /sys/class/net/*/device/sriov_numvfs )

for ifs in "${sriov_pfs[@]}"; do
  ifs_name="${ifs%%/device/*}"
  ifs_name="${ifs_name##*/}"
  ip link set "$ifs_name" netns "${CLUSTER_NAME}-control-plane"
done

# deploy multus
kubectl create -f $MANIFESTS_DIR/multus.yaml

#TODO check if this is still needed with the plugin
function configure-sriovdp() {
    local cmd_context="${1}" # context to run command e.g. sudo, docker exec
    ${cmd_context} "mkdir -p /etc/pcidp"
    ${cmd_context} "$(sriovdp-config-cmd)"
}

function sriovdp-config-cmd() {
    ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/sriovdp_setup.sh
    echo "cat <<EOF > /etc/pcidp/config.json
$(cat /etc/pcidp/config.json)
EOF
"
}

configure-sriovdp "${CONTROL_PLANE_CMD} bash -c"

# give them some time to create pods before checking pod status
sleep 10

# make sure all containers are ready
wait_containers_ready

${CONTROL_PLANE_CMD} chmod 666 /dev/vfio/vfio
${CONTROL_PLANE_CMD} mount -o remount,rw /sys     # kind remounts it as readonly when it starts, we need it to be writeable
