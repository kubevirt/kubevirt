# KubeVirt

KubeVirt is a virtual machine management architecture built around Kubernetes.

## Technical Overview

Kubernetes allows for extensions to its architecture in the form of 3rd party
resources: <http://kubernetes.io/docs/user-guide/thirdpartyresources/>.
KubeVirt represents virtual machines as 3rd party resources and manages changes
to libvirt domains based on the state of those resources.

This project provides a Vagrant setup with the requisite components already
installed. To boot a vanilla kubernetes environment as base for kubevirt,
simply type `vagrant up` from the root directory of the git tree, which can be
found here:
<!-- FIXME: <place URL to public git repository here> -->
Once the Vagrant provisioning script has completed, run `./cluster/sync.sh` to
build and deploy KubeVirt specific components to the Vagrant nodes.

Note: KubeVirt is built in go. A properly configured go environment is
therefore required. For best results, use this path:
`$GOPATH/src/kubevirt.io/kubevirt/`

### Associated resources

 * Kubernetes
 * Libvirt
 * Cockpit

### Project Components

 * virt-api: This component provides a HTTP RESTfull entrypoint to manage
   the virtual machines within the cluster.
 * virt-controller: This component manages the state of each VM within the
   Kubernetes cluster.
 * virt-handler: This is a daemon that runs on each Kubernetes node. It is
   responsible for monitoring the state of VMs according to Kubernetes and
   ensuring the corresponding libvirt domain is booted or halted accordingly.
 * virt-launcher: This component is a place-holder, one per running VM. Its
   job is to remain running as long as the VM is defined. This simply prevents a
   crash-loop state.
 * ha-proxy: This daemon proxies connections from 192.168.200.2 to the running
   master node--making it possible to establish connections in a consistent
   manner.

### Scripts

 * `cluster/sync.sh`: After deploying a fresh vagrant environment, or after
   making changes to code in this tree, this script will sync the Pods and
   DaemonSets in the running KubeVirt environment with the state of this tree.
 * `cluster/kubectl.sh`: This is a wrapper around Kubernete's kubectl command so
   that it can be run directly from this checkout without logging into a node.
 * `cluster/sync_config.sh`: This script will contact the master node and
   collect its config and kubectl. It is called by sync.sh so does not generally
   need to be run separately.

## Example

```
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

## Cockpit

Cockpit is exposed on <http://192.168.200.2:9090>
The default login is `root:vagrant`

It can be used to verify the running state of components within the cluster.
More information can be found on that project's site:

http://cockpit-project.org/guide/latest/feature-kubernetes.html

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
# TODO github repo here
git clone http://git.app.eng.bos.redhat.com/git/kubevirt/core.git $GOPATH/src/kubevirt.io/kubevirt
cd $GOPATH/src/kubevirt.io/kubevirt
```

To build the whole project, type

```bash
make
```

To build all docker images type

```bash
make docker
```

It is also possible to target only specific modules. For instance to build only
the `virt-controller`, type

```bash
make build WHAT=virt-controller
```

### Testing

Type

```bash
make test
```

to run all tests.

### Vagrant

Sets up a kuberentes cluster with a master and a node:

```bash
vagrant up
```

Build and deploy kubevirt:

```bash
bash cluster/sync.sh
```

Finally start a VM called `testvm`:

```bash
# this can be done from your GIT repo, no need to log into a vagrant VM
$ ./cluster/kubectl.sh create -f cluster/vm.json
```

This will start a VM on master or node with a macvtap and a tap networking
device attached.

Basic verification is possible by running

```
bash cluster/quickcheck.sh
```
