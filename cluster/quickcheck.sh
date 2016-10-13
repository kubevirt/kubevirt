#!/bin/bash

usage () {
echo "Usage: ./cluster/quickcheck.sh [-vm-name  <VM>] [-target-node <NODE>]"
echo "       ./cluster/quickcheck.sh [-clean-all]"
}

startvm () {
HEADERS="-H \"Content-Type: application/xml\""
VM_NAME=$1
TARGET_NODE=$2
DOMAIN="sed -e s/testvm/$VM_NAME/g cluster/testdomain.xml"
if [ "x" != "x$TARGET_NODE" ]; then
$DOMAIN | curl -X POST -H "Content-Type: application/xml" -H "Node-Selector: kubernetes.io/hostname=$TARGET_NODE" http://192.168.200.2:8182/api/v1/domain/raw -d @-
else
$DOMAIN | curl -X POST -H "Content-Type: application/xml" http://192.168.200.2:8182/api/v1/domain/raw -d @-
fi
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
vagrant ssh master -c "kubectl delete pods -l domain=${VM_NAME}"
set -e
sleep 2
startvm $VM_NAME $TARGET_NODE
sleep 10
NODE=$(vagrant ssh master -c "kubectl get pods -o json -l domain=${VM_NAME} | jq '.items[].spec.nodeName' -r" | sed -e 's/[[:space:]]*$//')

if [ -z $NODE ]; then
echo "Could not detect the VM."
exit 1
fi
echo "Found VM running on node '$NODE'"
# VM can also spawn on node
vagrant ssh $NODE -c "sudo /vagrant/cluster/verify-qemu-kube ${VM_NAME}"
else
vagrant ssh master -c "kubectl delete pods -l app=virt-launcher"
fi
