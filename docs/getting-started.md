# Getting Started

A quick start guide to get KubeVirt up and running inside Vagrant.

**Note**: This guide was tested on Fedora 23 and Fedora 25.

**Note:** Fedora 24 is known to have a bug which affects our vagrant setup.


## Building

### Go

[Go](https://golang.org) needs to be setup to be able to compile the sources.

**Note:** Go is pretty picky about paths, thus use the suggested ones.

```bash
    # If you haven't set it already, set a GOPATH
    echo "export GOPATH=~/go" >> ~/.bashrc
    echo "export PATH=$PATH:$GOPATH/bin" >> ~/.bashrc
    source ~/.bashrc

    mkdir -p ~/go

    sudo dnf install golang
```


### Vagrant

[Vagrant](https://www.vagrantup.com/) is used to bring up a development and
demo environment:

```bash
    sudo dnf install vagrant vagrant-libvirt
```

That's it for now with vagrant, it will be used further down.


### Build dependencies

Now we can finally get to the sources, before building KubeVirt we'll need
to install a few build requirements:

```bash
    # We are interfacing with libvirt
    sudo dnf install libvirt-devel

    sudo dnf install python-pip
    sudo pip install j2cli


    cd $GOPATH
    # First we setup govendor which is used to track dependencies
    go get -u github.com/kardianos/govendor
```

### Sources

Now we can clone the project into your `$GOPATH`:

```bash
    git clone https://github.com/kubevirt/kubevirt.git $GOPATH/src/kubevirt.io/kubevirt
    cd $GOPATH/src/kubevirt.io/kubevirt
```

And finally build all required artifacts and finally launch the
Vagrant environment:

```bash
    # Building and deploying kubevirt in Vagrant
    vagrant up
    cluster/sync.sh
```

This will create a VM called `master` which acts as Kubernetes master and then
deploy Kubevirt there. To create one or more nodes which will register
themselves on master, you can use the `VAGRANT_NUM_NODES` environment variable.
This would create a master and two nodes:

```bash
    VAGRANT_NUM_NODES=2 vagrant up
```

However, just running `master` is enough for most development tasks.

You could also run some build steps individually:

```bash
    # To build all binaries
    make

    # Or to build just one binary
    make build WHAT=virt-controller

    # To build all docker images
    make docker
```

### Code generation

**Note:** This is only important if you plan to modify sources, you don't need code generators just for building

Currently we use code generators for two purposes:

 * Generating swagger documentation out of struct and field comments for [go-restful](https://github.com/emicklei/go-restful)
 * Generating mock interfaces for [gomock](https://github.com/golang/mock)

So if you add or modify comments on structs in `pkg/api/v1` or if you change
interface definitions, you need to rerun the code generator.

First install the generator tools:

```bash
go get -u github.com/golang/mock/gomock
go get -u github.com/golang/mock/mockgen
go get -u github.com/rmohr/go-swagger-utils/swagger-doc
```

Then regenerate the code:

```bash
make generate
```

### Testing

After a succefull build you can run the testsuite using:

```bash
    make test
```


## Use

Congratulationsyou are still with us and you have build KubeVirt.

Now it's time to get hands on and give it a try.

### Cockpit

Cockpit is exposed on <http://192.168.200.2:9090>
The default login is `root:vagrant`

It can be used to view the cluster and verify the running state of
components within the cluster.
More information can be found on that [project's site](http://cockpit-project.org/guide/latest/feature-kubernetes.html).

### Create a first Virtual Machine

Finally start a VM called `testvm`:

```bash
    # This can be done from your GIT repo, no need to log into a vagrant VM
    # You might want to watch the Cockpit Cluster topology while running these commands

    # Create a VM
    ./cluster/kubectl.sh create -f cluster/vm.json

    # Sure? Let's list all created VMs
    ./cluster/kubectl.sh get vms

    # Enough, let's get rid of it
    ./cluster/kubectl.sh delete -f cluster/vm.json


    # You can actually use kubelet.sh to introspect the cluster in general
    ./cluster/kubectl.sh get pods
```

This will start a VM on master or one of the running nodes with a macvtap and a
tap networking device attached.

Basic verification is possible by running

```bash
    bash cluster/quickcheck.sh
```

#### Example

```bash
$ ./cluster/kubectl.sh create -f cluster/vm.json
vm "testvm" created

$ ./cluster/kubectl.sh get pods
NAME                        READY     STATUS    RESTARTS   AGE
haproxy                     1/1       Running   4          10h
virt-api                    1/1       Running   1          10h
virt-controller             1/1       Running   1          10h
virt-handler-z90mp          1/1       Running   1          10h
virt-launcher-testvm9q7es   1/1       Running   0          10s

$ ./cluster/kubectl.sh get vms
NAME      LABELS                        DATA
testvm    kubevirt.io/nodeName=master   {"apiVersion":"kubevirt.io/v1alpha1","kind":"VM","...

$ ./cluster/kubectl.sh get vms -o json
{
    "kind": "List",
    "apiVersion": "v1",
    "metadata": {},
    "items": [
        {
            "apiVersion": "kubevirt.io/v1alpha1",
            "kind": "VM",
            "metadata": {
                "creationTimestamp": "2016-12-09T17:54:52Z",
                "labels": {
                    "kubevirt.io/nodeName": "master"
                },
                "name": "testvm",
                "namespace": "default",
                "resourceVersion": "102534",
                "selfLink": "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm",
                "uid": "7e89280a-be62-11e6-a69f-525400efd09f"
            },
            "spec": {
    ...
```

### Accessing the VM via Spice

First make sure you have `remote-viewer` installed. On Fedora run

```bash
dnf install virt-viewer
```

Then, after you made sure that the VM `testvm` is running, type

```
cluster/kubectl.sh spice testvm
```

to start a remote session with `remote-viewer`.

To print the connection details to stdout, run

```bash
cluster/kubectl.sh spice testvm --details
```
