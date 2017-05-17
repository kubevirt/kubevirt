#!/bin/bash
set -ex

kubectl() { cluster/kubectl.sh --core "$@"; }


# Install GO
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=1.7.4 bash)"
export GOPATH=$PWD/go
export GOBIN=$PWD/go/bin
export PATH=$GOPATH/bin:$PATH
export VAGRANT_NUM_NODES=1

# Install dockerize
export DOCKERIZE_VERSION=v0.3.0
curl -LO https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

# Needed for templating of manifests
pip install j2cli

# Keep .vagrant files between builds
export VAGRANT_DOTFILE_PATH=~/.vagrant

# Install vagrant-libvirt plugin if not present
vagrant plugin install vagrant-libvirt

# Go to the project directory with the sources and the Vagrantfile
cd $GOPATH/src/kubevirt.io/kubevirt

# Make sure that the VM is properly shut down on exit
trap '{ vagrant halt; }' EXIT

set +e
vagrant up --provider=libvirt
if [ $? -ne 0 ]; then
  # After a workspace cleanup we loose our .vagrant file, this means that we have to clean up libvirt
  vagrant destroy
  virsh undefine kubevirt_master
  virsh destroy kubevirt_master
  virsh undefine kubevirt_node0
  virsh destroy kubevirt_node0
  virsh net-destroy vagrant0
  virsh net-undefine vagrant0
  # Remove now stale images
  vagrant destroy
  set -e
  vagrant up --provider=libvirt
fi
set -e

# Build kubevirt
go get -u github.com/kardianos/govendor
make

# Copy connection details for kubernetes
cluster/kubectl.sh --init

# Make sure we can connect to kubernete
export APISERVER=$(cat cluster/vagrant/.kubeconfig | grep server | sed -e 's# \+server: https://##' | sed -e 's/\r//')
/usr/local/bin/dockerize -wait tcp://$APISERVER -timeout 120s

# Wait for nodes to become ready
while [ -n "$(kubectl get nodes --no-headers | grep -v Ready)" ]; do
   echo "Waiting for all nodes to become ready ..."
   kubectl get nodes --no-headers | >&2 grep -v Ready
   sleep 10
done
echo "Nodes are ready:"
kubectl get nodes

echo "Work around bug https://github.com/kubernetes/kubernetes/issues/31123: Force a recreate of the discovery pods"
kubectl delete pods -n kube-system -l k8s-app=kube-discovery
sleep 10
while [ -n "$(kubectl get pods -n kube-system --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubernetes pods to become ready ..."
    kubectl get pods -n kube-system --no-headers | >&2 grep -v Running
    sleep 10
done

echo "Kubernetes is ready:"
kubectl get pods -n kube-system
echo ""
echo ""

# Delete traces from old deployments
kubectl delete deployments --all
kubectl delete pods --all

# Deploy kubevirt
cluster/sync.sh

# Wait until kubevirt is ready
while [ -n "$(kubectl get pods --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubevirt pods to become ready ..."
    kubectl get pods --no-headers | >&2 grep -v Running
    sleep 10
done
kubectl get pods
cluster/kubectl.sh version

until timeout 10 cluster/kubectl.sh create -f cluster/vm.yaml
do
  echo "Creating the VM test deployment failed, will retry in 10 seconds ..."
  sleep 10
done

while [ -z "$(timeout 10 cluster/kubectl.sh get vms testvm -o yaml | grep Running)" ]; do
    echo "Waiting for the VM test deployment to become ready ..."
    sleep 10
done

echo "VM test deployment successfully created."

# Run functional tests
cluster/run_tests.sh --ginkgo.noColor
