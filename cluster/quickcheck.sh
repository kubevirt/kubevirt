#!/bin/bash

source hack/config.sh
usage () {
echo "Usage: ./cluster/quickcheck.sh [-vm-name  <VM>] [-target-node <NODE>]"
echo "       ./cluster/quickcheck.sh [-clean-all]"
}

startvm () {
VM_NAME=$1
# TODO fix target node
TARGET_NODE=$2
DOMAIN="sed -e s/testvm/$VM_NAME/g cluster/vm.json"
$DOMAIN | cluster/kubectl.sh create -f -
}

VM_NAME=testvm

while [[ $# -gt 0 ]]
do
key="$1"


case $key in
    -vm-name)
        VM_NAME="$2"
        shift
    ;;
    -clean-all)
        CLEAN_ALL="true"
    ;;
    -target-node)
        TARGET_NODE=$2
        shift
    ;;
    -help)
        usage
        exit 0
    ;;

    *)
        usage
        exit 1
    ;;
esac
shift
done

JQ_PRESENT=$(which jq >/dev/null 2>&1 && echo 1 || echo 0)
if [ "$JQ_PRESENT" == "0" ]
then
    echo "Missing required system dependency: jq"
    exit 1
fi

if [ -z "$CLEAN_ALL" ]; then
  # Delete old VM if it exists
  cluster/kubectl.sh --core delete pods -l kubevirt.io/domain=${VM_NAME}
  # TODO only do this delete when it exists, to avoid misleading stderr output
  cluster/kubectl.sh --core delete vms ${VM_NAME}
  set -e
  sleep 2
  # Start new VM
  startvm $VM_NAME $TARGET_NODE
  sleep 10
  # Try to detect the node where the VM was scheduled to
  NODE=$(cluster/kubectl.sh --core get pods -o json -l kubevirt.io/domain=${VM_NAME} | jq '.items[].spec.nodeName' -r)

  if [ -z $NODE ]; then
    echo "Could not detect the VM."
    exit 1
  fi
  echo "Found VM running on node '$NODE'"
  # Verify that the VM is running and in the right cgroups and namespaces
  vagrant ssh $NODE -c "sudo /vagrant/cluster/verify-qemu-kube ${VM_NAME}"
else
  # Remove all VMs and VM pods
  cluster/vagrant/kubectl.sh --core delete pods -l kubevirt.io/app=virt-launcher
  cluster/vagrant/kubectl.sh --core delete vms --all
fi
