#!/bin/bash

source hack/config.sh
usage () {
echo "Usage: ./cluster/quickcheck.sh [-vm  <VM>]"
}

VM_NAME=testvm

while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -vm)
        VM_NAME="$2"
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

NODE=$(cluster/kubectl.sh --core get pods -o json -l kubevirt.io/domain=${VM_NAME} | jq '.items[].spec.nodeName' -r)

if [ -z $NODE ]; then
  echo "Could not detect the VM."
  exit 1
fi
echo "Found VM running on node '$NODE'"
# Verify that the VM is running and in the right cgroups and namespaces
vagrant ssh $NODE -c "sudo /vagrant/cluster/verify-qemu-kube ${VM_NAME}"
