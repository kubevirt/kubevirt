# KubeVirt

## Hacking

### Setup

First make sure you have [govendor](https://github.com/kardianos/govendor)
installed.

To install govendor in your `$GOPATH/bin` simply run

```bash
go get -u github.com/kardianos/govendor
```

If you don't have the `$GOPATH/bin` folder on your path, do

```bash
export PATH=$PATH:$GOPATH/bin
```

### Building

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
make all contrib
vagrant rsync # if you do not use NFS
vagrant ssh master -c "cd /vagrant && sudo hack/build-docker.sh"
vagrant ssh node -c "cd /vagrant && sudo hack/build-docker.sh"
vagrant ssh master -c "kubectl create -f /vagrant/contrib/manifest/virt-controller.yaml"
vagrant ssh master -c "kubectl create -f /vagrant/contrib/manifest/virt-controller-service.yaml"
```

Finally start a VM called `testvm`:

```bash
# this can be done from outside the VMs vecause of the virt-controller-service
curl -X POST -H "Content-Type: application/xml" http://192.168.200.2:8182/api/v1/domain/raw -d @pkg/virt-launcher/domain.xml  -v
```
