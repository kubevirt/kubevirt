# Storing VM Disks in the Container Registry

## Motivation

KubeVirt is a drop in solution for running VMs on an existing kubernetes
cluster. One of our goals in doing this is to leverage the tools and workflows
kubernetes has established for containers and apply them to virtualized
workloads.

For example, with KubeVirt VMs are managed like pods. Users post VMs to
kubernetes like pods and the VMs will soon have the option to integrate into
the kubernetes overlay network like pods.

To continue down the path of allowing users to introduce VM workloads into
an existing kubernetes cluster with as little friction as possible, the next
hurdle for us to overcome is tying VM disk distribution into the same
mechanism used by pods. To do this, KubeVirt must have a way of storing and
accessing VM disks using a container registry.

## Requirements
* Users must have a workflow for pushing VM disks into a container registry.
* KubeVirt runtime must be able to consume VM disks stored in the container
  registry and attach those disks to VMs.

## Local Ephemeral Disk Use Cases

### Immutable VMs booting from ephemeral disk

Users want to launch VM workloads backed by local ephemeral storage. The VM
workload does not need to remain persistent across VM restarts and the workload
does not require live migration support.

### Immutable VM Replication booting from ephemeral local disks.

Users want to horizontally scale VM workloads backed by local ephemeral
storage. *Pending introduction and implementation of VMGroup object*

## Ephemeral Non Bootable local disk storage

Users want to attach a non-bootable disk to a VM preloaded with data. This data
could consist of anything. It could be configuration data, rpms, or anything
else a user might want to distribute with their VMs.

## Implementation

### High Level Design

**Standardised KubeVirt VM disk Wrapper**

KubeVirt provides a standardized base wrapper container image that serves up a
user provided VM disk as a local file consumable by Libvirt. This base
container image does not have a VM disk inside of it, it merely contains
everything necessary for serving up a VM disk in a way Libvirt can consume.

**User Workflow**

Users push VM Images into the container registry using the KubeVirt base
wrapper container image. By default, the base wrapper container KubeVirt
provides will serve up any VM disk placed in the /disk directory as a block
device consumable by libvirt.

Example:
```
cat << END > Dockerfile
FROM kubevirt.io:disk
ADD fedora25.qcow2 /disk
END
docker build -t vmdisks/fedora25:latest .
docker push vmdisks/fedora25:latest
```
Users start a VM using a "container registry backed VM disk" by referencing the
image name they pushed to the registry. Any number of VMs can be started
anywhere in the cluster referencing the same image.

Example:
```
kind: VirtualMachine
spec:
  domain:
    devices:
      disks:
      - type: RegistryDisk:v1alpha
        - source:
          name: vmdisks/fedora25:latest
        - target:
          device: sda
```

**KubeVirt Runtime Implementation**

When virt-controller sees a VM object with a disk type of
**RegistryDisk:v1alpha**, virt-controller places the image wrapper container
in the virt-launcher pod along with the container that monitors the VM process.

When the virt-launcher pod starts, the wrapper container copies the user's VM
image as file on a shared host directory.

When virt-handler sees a VM is placed on the local node with the
**RegistryDisk:v1alpha** disk type defined, virt-handler injects the
necessary configuration into the domain xml required to add the disk backed by
a local file.

The **v1** part of the RegistryDisk disk type represents the standard
used during the virt-handler disk conversion process. As we gain more
experience with this feature, we may want to adopt a new standard for how VM
images are wrapped by a container while maintaining backwards compatibility.
