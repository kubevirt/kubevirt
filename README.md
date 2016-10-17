# KubeVirt

## Hacking

Before you start coding, the [Project structure overview](docs/structure.md)
should help you understanding the project and the microservices layout.

### Setup

First make sure you have [govendor](https://github.com/kardianos/govendor),
`j2cli` and `libvirt-devel` installed.

To install govendor in your `$GOPATH/bin` simply run

```bash
go get -u github.com/kardianos/govendor
```

If you don't have the `$GOPATH/bin` folder on your path, do

```bash
export PATH=$PATH:$GOPATH/bin
```

`j2cli` can be installed with

```bash
sudo pip install j2cli
```

On Fedora `libvirt-devel` can be  installed with

```bash
sudo dnf install libvirt-devel
```

### Building

First clone the project into your `$GOPATH`:

```bash
git clone http://git.app.eng.bos.redhat.com/git/kubevirt/core.git $GOPATH/src/kubevirt/core
cd $GOPATH/src/kubevirt/core
```

To build the whole project, type

```bash
make
```

To build all docker images type

```bash
make docker
```

It is also possible to target only specific modules. For instance to build only the `virt-launcher`, type

```bash
make build WAHT=virt-launcher
```

### Testing

Type

```bash
make test
```

to run all tests.

### Vagrant

TODO, IMPROVE THAT FLOW:

Sets up a kuberentes cluster with a master and a node:
```bash
# export VAGRANT_USE_NFS=true # if you want to use nfs
vagrant up
```

Build and deploy kubevirt:

```bash
bash cluster/sync.sh
```

Finally start a VM called `testvm`:

```bash
# this can be done from outside the VMs vecause of the virt-controller-service
curl -X POST -H "Content-Type: application/xml" http://192.168.200.2:8182/api/v1/domain/raw -d @cluster/testdomain.xml
```

This will start a VM on master or node with a macvtap and a tap networking device attached.

Basic verifcation is possible by running

```
# TODO, there is an issue with the detection of where the VM is sceduled
bash cluster/quicktest.sh
```
