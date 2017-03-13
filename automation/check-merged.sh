set -ex

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

# Make sure we can connect to kubernetes pods are running
export APISERVER=$(cat cluster/vagrant/.kubeconfig | grep server | sed -e 's# \+server: https://##' | sed -e 's/\r//')
/usr/local/bin/dockerize -wait tcp://$APISERVER -timeout 120s
while [ -n "$(cluster/kubectl.sh --core get pods --namespace kube-system --no-headers | grep -v Running)" ]; do sleep 10; done
cluster/kubectl.sh --core get pods --namespace kube-system

# Delete traces from old deployments
cluster/kubectl.sh --core delete deployments --all
cluster/kubectl.sh --core delete pods --all
cluster/kubectl.sh --core delete jobs --all

# Deploy kubevirt
cluster/sync.sh

# Wait until kubevirt is ready
while [ -n "$(cluster/kubectl.sh --core get pods --no-headers | grep -v Running)" ]; do sleep 10; done
cluster/kubectl.sh --core get pods
cluster/kubectl.sh version

# Run functional tests
cluster/run_tests.sh --ginkgo.noColor
