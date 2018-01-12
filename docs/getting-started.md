# Getting Started

A quick start guide to get KubeVirt up and running inside Vagrant.

**Note**: This guide was tested on Fedora 23 and Fedora 25.

**Note:** Fedora 24 is known to have a bug which affects our vagrant setup.

## Building

The KubeVirt build system runs completely inside docker. In order to build
KubeVirt you need to have `docker` and `rsync` installed.

### Vagrant

[Vagrant](https://www.vagrantup.com/) is used to bring up a development and
demo environment:

```bash
    sudo dnf install vagrant vagrant-libvirt
    sudo systemctl enable --now libvirtd
    sudo systemctl restart virtlogd # Work around rpm packaging bug
```

On some systems Vagrant will always ask you for your sudo password when you try
to do something with a VM. To avoid retyping your password all the time you can
add yourself to the `libvirt` group.

```bash
sudo gpasswd -a ${USER} libvirt
newgrp libvirt
```

On CentOS/RHEL 7 you might also need to change the libvirt connection string to be able to see all libvirt information:

```
export LIBVIRT_DEFAULT_URI=qemu:///system
```

### Compile and run it

Build all required artifacts and launch the
Vagrant environment:

```bash
    # Building and deploying kubevirt in Vagrant
    make cluster-up
    make cluster-sync
```

This will create a VM called `master` which acts as Kubernetes master and then
deploy Kubevirt there. To create one or more nodes which will register
themselves on master, you can use the `VAGRANT_NUM_NODES` environment variable.
This would create a master and one node:

```bash
    VAGRANT_NUM_NODES=1 vagrant up
```

If you decide to use separate nodes, pass `VAGRANT_NUM_NODES` variable to all
vagrant interacting commands. However, just running `master` is enough for most
development tasks.

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

They don't require vagrant. To run the *functional tests*, make sure you have set
up [Vagrant](#vagrant). Then run

```bash
    make cluster-sync # synchronize with your code, if necessary
    make functest # run the functional tests against the Vagrant VMs
```

## Use

Congratulations you are still with us and you have build KubeVirt.

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

    # Create a VM
    ./cluster/kubectl.sh create -f cluster/vm.yaml

    # Sure? Let's list all created VMs
    ./cluster/kubectl.sh get vms

    # Enough, let's get rid of it
    ./cluster/kubectl.sh delete -f cluster/vm.yaml


    # You can actually use kubelet.sh to introspect the cluster in general
    ./cluster/kubectl.sh get pods
```

This will start a VM on master or one of the running nodes with a macvtap and a
tap networking device attached.

#### Example

```bash
$ ./cluster/kubectl.sh create -f cluster/vm.json
vm "testvm" created

$ ./cluster/kubectl.sh get pods
NAME                        READY     STATUS    RESTARTS   AGE
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
            "kind": "VirtualMachine",
            "metadata": {
                "creationTimestamp": "2016-12-09T17:54:52Z",
                "labels": {
                    "kubevirt.io/nodeName": "master"
                },
                "name": "testvm",
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

Then, after you made sure that the VM `testvm` is running, type

```
cluster/kubectl.sh vnc testvm
```

to start a remote session with `remote-viewer`.

Since `kubectl` does not support TPR subresources yet, the above `cluster/kubectl.sh vnc` magic is just a wrapper.
