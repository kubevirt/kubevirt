#!/bin/bash -xe
source hack/config

curl -L -k "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/virtctl-${KUBEVIRT_VERSION}-linux-amd64" --output virtctl
chmod +x virtctl

export KUBEVIRT_PROVIDER=${TARGET:-"k8s-1.17"}

kubectl() { cluster/kubectl.sh "$@"; }

make cluster-down
KUBEVIRT_NUM_NODES=3 make cluster-up

# Make sure all the workers are tagged
WORKER_NODES=$(kubectl get nodes -o json | jq -r '.items[] | select(.metadata.labels["node-role.kubernetes.io/master"] == null) | .metadata.name')
for node in $WORKER_NODES; do
  kubectl label node "${node}" node-role.kubernetes.io/worker=""
done

# Label infra node and workloads node
# shellcheck disable=SC2206
WORKERS_ARR=(${WORKER_NODES})
kubectl label node "${WORKERS_ARR[0]}" node.kubernetes.io/instance-type=infra
kubectl label node "${WORKERS_ARR[1]}" node.kubernetes.io/instance-type=workloads

HCO_CONFIGURATION_HOOK=hack/np-config-hook.sh make cluster-sync

WORKLOADS_NODE=$(kubectl get node -l "node.kubernetes.io/instance-type=workloads" -o name)
WORKLOADS_NODE="${WORKLOADS_NODE#*/}"
[[ "${WORKLOADS_NODE}" == "${WORKERS_ARR[1]}" ]]

set -x
for i in {1..15}; do
  echo "run number $i"
  kubectl apply -f hack/testvm.yaml
  ./virtctl start testvm

  for _ in {1..15}; do
    VMI_NODE=$(kubectl get vmi testvm -o json | jq -r '.metadata.labels["kubevirt.io/nodeName"]')
    if [[ "${VMI_NODE}" != "null" ]]; then
      break 1
    fi
    sleep 20
  done
  if [[ "${VMI_NODE}" != "${WORKLOADS_NODE}" ]]; then
    # exit 1
    echo error
    break
  fi
  ./virtctl stop testvm
  kubectl wait vmi testvm --for delete --timeout=60s
  kubectl delete -f hack/testvm.yaml
done


