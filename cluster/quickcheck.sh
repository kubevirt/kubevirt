#!/bin/bash
set -xe

vagrant ssh master -c 'kubectl delete pods -l domain=testvm'
sleep 2
vagrant ssh master -c 'curl -X POST -H "Content-Type: application/xml" http://192.168.200.2:8182/api/v1/domain/raw -d @/vagrant/cluster/testdomain.xml'
sleep 10
# TODO this gives me the output "$'master'\r" instead of plain "master"
#NODE=$(vagrant ssh master -c "kubectl get pods -o json -l domain=testvm | jq '.items[].spec.nodeName' -r")
NODE=master

# VM can also spawn on node
vagrant ssh $NODE -c "sudo /vagrant/cluster/verify-qemu-kube testvm"
