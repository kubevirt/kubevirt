set -ex

# Install GO
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=1.7.4 bash)"
export GOPATH=$PWD/go
export GOBIN=$PWD/go/bin
export PATH=$GOPATH/bin:$PATH

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

# Run unit tests
# make test

# Delete traces from old deployments
cluster/kubectl.sh --init
cluster/kubectl.sh --core delete deployments --all
cluster/kubectl.sh --core delete pods --all

# Deploy kubevirt
cluster/sync.sh

# Wait until virt-api is ready
sleep 30
cluster/kubectl.sh --core get pods
cluster/kubectl.sh version

# Run functional tests
cluster/run_tests.sh --ginkgo.noColor
