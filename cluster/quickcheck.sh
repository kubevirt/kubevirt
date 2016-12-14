#!/bin/bash

source hack/config.sh

usage () {
echo "Usage: ./cluster/quickcheck.sh [-vm-name  <VM>] [-target-node <NODE>]"
echo "       ./cluster/quickcheck.sh [-clean-all]"
}

startvm () {
HEADERS="-H \"Content-Type: application/json\""
VM_NAME=$1
TARGET_NODE=$2
DOMAIN="sed -e s/testvm/$VM_NAME/g cluster/vagrant/vm.json"
$DOMAIN | curl -X POST -H "Content-Type: application/json" http://${master_ip}:8183/apis/kubevirt.io/v1alpha1/namespaces/default/vms -d @-
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

if [ -z "$CLEAN_ALL" ]; then
  vagrant ssh master -c "kubectl delete pods -l kubevirt.io/domain=${VM_NAME}"
  vagrant ssh master -c "kubectl delete vms ${VM_NAME}"
  set -e
  sleep 2
  startvm $VM_NAME $TARGET_NODE
  sleep 10
  NODE=$(vagrant ssh master -c "kubectl -s 127.0.0.1:8080 get pods -o json -l kubevirt.io/domain=${VM_NAME} | jq '.items[].spec.nodeName' -r" | sed -e 's/[[:space:]]*$//')

  if [ -z $NODE ]; then
    echo "Could not detect the VM."
    exit 1
  fi
  echo "Found VM running on node '$NODE'"
  # VM can also spawn on node
  vagrant ssh $NODE -c "sudo /vagrant/cluster/verify-qemu-kube ${VM_NAME}"
else
  vagrant ssh master -c "kubectl delete pods -l kubevirt.io/app=virt-launcher"
  vagrant ssh master -c "kubectl delete vms --all"
fi
