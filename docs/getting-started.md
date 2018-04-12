# Getting Started

A quick start guide to get KubeVirt up and running inside Vagrant.

## Building

The KubeVirt build system runs completely inside docker. In order to build
KubeVirt you need to have `docker` and `rsync` installed.

### Dockerizied environment

Runs master and nodes containers, when each one of them run virtual machine via QEMU.
In additional it runs dnsmasq and docker registry containers.

### Compile and run it

Build all required artifacts and launch the
dockerizied environment:

```bash
    # Building and deploying kubevirt in Vagrant
    export PROVIDER=k8s-1.9.3
    make cluster-up
    make cluster-sync
```

This will create a VM called `node01` which acts as Kubernetes master and then
deploy KubeVirt there. To create one or more nodes which will register
themselves on master, you can use the `VAGRANT_NUM_NODES` environment variable.
This would create a master and one node:

```bash
    export VAGRANT_NUM_NODES=1
    make cluster-up
```

You could also run some build steps individually:

```bash
    # To build all binaries
    make

    # Or to build just one binary
    make build WHAT=cmd/virt-controller

    # To build all docker images
    make docker
```

### Code generation

**Note:** This is only important if you plan to modify sources, you don't need code generators just for building

To invoke all code-generators and regenerate generated code, run:

```bash
make generate
```

### Testing

After a successful build you can run the *unit tests*:

```bash
    make test
```

They don't real environment. To run the *functional tests*, make sure you have set
up dockerizied environment. Then run

```bash
    make cluster-sync # synchronize with your code, if necessary
    make functest # run the functional tests against the Vagrant VMs
```

## Use

Congratulations you are still with us and you have build KubeVirt.

Now it's time to get hands on and give it a try.

### Create a first Virtual Machine

Finally start a VM called `vm-ephemeral`:

```bash
    # This can be done from your GIT repo, no need to log into a vagrant VM

    # Create a VM
    ./cluster/kubectl.sh create -f cluster/examples/vm-ephemeral.yaml

    # Sure? Let's list all created VMs
    ./cluster/kubectl.sh get vms

    # Enough, let's get rid of it
    ./cluster/kubectl.sh delete -f cluster/examples/vm-ephemeral.yaml


    # You can actually use kubelet.sh to introspect the cluster in general
    ./cluster/kubectl.sh get pods
```

This will start a VM on master or one of the running nodes with a macvtap and a
tap networking device attached.

#### Example

```bash
$ ./cluster/kubectl.sh create -f cluster/examples/vm-ephemeral.yaml
vm "vm-ephemeral" created

$ ./cluster/kubectl.sh get pods
NAME                              READY     STATUS    RESTARTS   AGE
virt-api                          1/1       Running   1          10h
virt-controller                   1/1       Running   1          10h
virt-handler-z90mp                1/1       Running   1          10h
virt-launcher-vm-ephemeral9q7es   1/1       Running   0          10s

$ ./cluster/kubectl.sh get vms
NAME           LABELS                        DATA
vm-ephemera    kubevirt.io/nodeName=node01   {"apiVersion":"kubevirt.io/v1alpha1","kind":"VM","...

$ ./cluster/kubectl.sh get vms -o json
{
    "kind": "List",
    "apiVersion": "v1",
    "metadata": {},
    "items": [
        {
            "apiVersion": "kubevirt.io/v1alpha1",
            "kind": "VirtualMachine",
            "metadata": {
                "creationTimestamp": "2016-12-09T17:54:52Z",
                "labels": {
                    "kubevirt.io/nodeName": "master"
                },
                "name": "vm-ephemeral",
                "namespace": "default",
                "resourceVersion": "102534",
                "selfLink": "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm",
                "uid": "7e89280a-be62-11e6-a69f-525400efd09f"
            },
            "spec": {
    ...
```

### Accessing the Domain via VNC

First make sure you have `remote-viewer` installed. On Fedora run

```bash
dnf install virt-viewer
```

Then, after you made sure that the VM `vm-ephemeral` is running, type

```
cluster/kubectl.sh vnc vm-ephemeral
```

to start a remote session with `remote-viewer`.

Since `kubectl` does not support TPR subresources yet, the above `cluster/kubectl.sh vnc` magic is just a wrapper.
