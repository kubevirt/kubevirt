#!/bin/bash
set -x

vagrant ssh master -c 'kubectl delete pods -l domain=testvm'
sleep 2
vagrant ssh master -c 'curl -X POST -H "Content-Type: application/xml" http://192.168.200.2:8182/api/v1/domain/raw -d @/vagrant/cluster/testdomain.xml -v'
sleep 10
NODE=$(vagrant ssh master -c "kubectl get pods -o json -l domain=testvm | jq '.items[].spec.nodeName' -r" | sed -e 's/[[:space:]]*$//')

if [ -z $NODE ]; then
echo "Could not detect the VM."
exit 1
fi
echo "Found VM running on node $NODE"
# VM can also spawn on node
vagrant ssh $NODE -c "sudo /vagrant/cluster/verify-qemu-kube testvm"
