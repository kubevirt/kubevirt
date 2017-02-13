set -ex

# Install GO
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=1.7.4 bash)"
export GOPATH=$PWD/go
export GOBIN=$PWD/go/bin
export PATH=$GOPATH/bin:$PATH

# Needed for templating of manifests
pip install j2cli

# Use dockerize to detect if services are ready
export DOCKERIZE_VERSION=v0.3.0
wget -q https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz -O dockerize.tar.gz
tar -C /usr/local/bin -xzvf dockerize.tar.gz

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

# Deploy kubevirt
cluster/sync.sh

# Wait until virt-api is ready
#/usr/local/bin/dockerize -wait http://192.168.200.2:8183/apis/kubevirt.io/v1alpha1/healthz -timeout 240s
sleep 30
cluster/kubectl.sh --core version
cluster/kubectl.sh --core get pods
cluster/kubectl.sh version

# Run functional tests
cluster/run_tests.sh
