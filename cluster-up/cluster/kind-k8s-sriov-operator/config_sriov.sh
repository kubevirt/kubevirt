#!/bin/bash -e
set -x

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

# give them some time to create pods before checking pod status
sleep 10

# make sure all containers are ready
wait_containers_ready

${CONTROL_PLANE_CMD} mount -o remount,rw /sys     # kind remounts it as readonly when it starts, we need it to be writeable

git clone https://github.com/openshift/sriov-network-operator.git
pushd sriov-network-operator
make deploy-setup
popd

kubectl label node sriov-control-plane node-role.kubernetes.io/worker=
kubectl label node sriov-control-plane sriov=true 
kubectl wait --for=condition=Ready pod --all -n sriov-network-operator --timeout 6m

sleep 5 #give the operator the time to spin the node config daemon
kubectl wait --for=condition=Ready pod --all -n sriov-network-operator --timeout 6m

# TO BE FIXED IN SRIOV OPERATOR
NETWORK_DAEMON_POD=$(kubectl get pods -n sriov-network-operator | grep sriov-network-config-daemon | awk '{print $1}')
kubectl exec -n sriov-network-operator $NETWORK_DAEMON_POD -- bash -c "cat >bindata/scripts/enable-kargs.sh <<EOL
#!/bin/bash
set -x
echo 0
EOL
"

kubectl create -f $MANIFESTS_DIR/network_policy.yaml
sleep 10 #let the cni daemon appear
kubectl wait --for=condition=Ready pod --all -n sriov-network-operator --timeout 12m

${CONTROL_PLANE_CMD} chmod 666 /dev/vfio/vfio
# TO BE FIXED IN SRIOV OPERATOR
${CONTROL_PLANE_CMD} cp /var/lib/cni/bin/sriov /opt/cni/bin/

