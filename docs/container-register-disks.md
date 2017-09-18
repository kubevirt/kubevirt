# Storing VM Disks in the Container Registery

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
hurdle for us to overcome is tieing VM disk distribution into the same
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

## Future Use Cases
For the sake of simplicity, this proposal is focusing on the details involved
with use cases where Immutable VMs are launched using ephemeral storage that is
not persistent between VM restarts.

Container Registry backed disks can be extended work with stateful block
storage. This persistent block storage functionality will be built upon the
foundation laid by this initial proposal.

High level notes related to how container registry disks integrate with
persistent block storage is provided at the end of this document.

## Implementation 

### Concepts

When a container starts, the container's image is pulled from a remote
container registry. Each container image consists of multiple layers and only
the layers that are not already present locally are retrieved from the remote
registry. 

When a container is being initialized, a union mount filesystem is used to
combine all the container image's layers together to form the container's
directory tree. When processes in a container modify contents on the filesystem,
the processes are not modifying the layers provided by the container image, they
are modifying a layer of the union filesystem that is tracking the delta between
the active container and the underlying layers provided by the container image.

The delta between the container image and an active container is ephemeral and
is automatically freed when container is destroyed. Depending on the union
mount filesystem implementation used, the delta may be tracked at the block
level or the file level. When tracked at the file level, the first time a file
is written to causes the entire file to be copied into the delta layer.

These are the concepts we're building upon to provide an ephemeral disk to a VM
backed backed by a container image.

### High Level Design

**Standardised KubeVirt VM disk Wrapper**

KubeVirt provides a standardized base wrapper container image that serves up a
user provided VM disk as a network disk consumable by Libvirt. This base
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
*NOTE: there's more info on how qcow2 is served as a block device later*

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
      - type: ContainerRegistryDisk:v1alpha
        - source:
          name: vmdisks/fedora25:latest
        - target:
          device: sda
```

**KubeVirt Runtime Implementation**

When virt-controller sees a VM object with a disk type of
**ContainerRegistryDisk:v1alpha**, virt-controller places the image wrapper container
in the virt-launcher pod along with the container that monitors the VM process.

When the virt-launcher pod starts, the wrapper container serves up the user's VM
image as an network block device Libvirt can consume. The logic that serves up
the vm image as a block device is what we provide in the standardized KubeVirt
base wrapper container.

When virt-handler sees a VM is placed on the local node with the
**ContainerRegistryDisk:v1alpha** disk type defined, virt-handler injects the
necessary configuration into the domain xml file required to connect to the
block device served up by the wrapper container.

The **v1** part of the ContainerRegistryDisk disk type represents the standard
used during the virt-handler disk conversion process. As we gain more
experience with this feature, we may want to adopt a new standard for how VM
images are wrapped by a container while maintaining backwards compatibility. 

**Bringing it all together**
* KubeVirt provides a base container that serves up VM disks as a network
  block device libvirt can consume
* Users push their VM disks into the container registry using the base image
* Users reference VM disks backed by the container registry with the
  **ContainerRegistryDisk:v1alpha** disk type.
* The virt-launcher pod has an instance of the wrapper container which serves
  up the VM disk as an ephemeral block device.
* virt-handler sees the ContainerRegistryDisk:v1alpha disk type and automatically
  injects disk configuration into the domain xml that gives the VM access to the
  block device served up by the wrapper container.
* Anything the VM writes to disk is destroyed when when the VM is torn down as a
  result of the virt-launcher pod (which has the wrapper container) being
  destroyed.

### v1 Base Container Design

**iscsi Base Container**

The **v1** base container will serve up a VM disk as a block device using
iscsi. Conceptually, this base container will do the equivalent of what the
images/iscsi-demo-target-tgtd does today. The big difference here is no actual
VM disk will be present in the base container, only the dependencies and
entry point script required to serve up a user provided image.

**Recommended qcow2 Usage**

Although we don't have to enforce this, It is recommended that users upload
their images in qcow2 format.

This is important for two reasons.
1. VMs with large disks can be stored in the container registry while only
   consuming the minimal amount of network storage.
2. The most common union mount filesystems do not act on block storage, but
   instead copy up on file writes. To avoid the copy up performance penalty,
   the wrapper container will have its own unique copy of the VM disk. Using
   qcow2 will reduce the local disk space required. The container image will
   only need the qcow2 file, while the actual running wrapper container will
   expand that image to raw format.

Example of the logic the wrapper container will use to expand a qcow2 image to
a raw image that can be served as a block device. The qcow2 image could be 1gb
which could expand to a 100gb ephemeral raw file.
```
qemu-img convert image.qcow2 image.raw
```

**Authentication**

The KubeVirt runtime will handle automatically generating credentials required
for iscsi CHAP authentication. The credentials will be passed to the image
wrapper container via environment variables.  Virt-handler will coordinate
injecting these credentials into libvirt allowing the VM access to the container
backed disk.

# The Future: Persistent Disks, Live Migrations and Flying Cars.
This proposal is meant to provide a starting point for introducing the concept
of storing and accessing VM disks in a container registry. To reduce complexity
the scope has been reduced to ephemeral disks for the initial work.

However, this concept can be extended to provide persistent disks and support
live migrations by using PersistentVolumeClaims in conjunction with the VM Image
wrapper container.

**Persistent Disks**

Persistent disks can be supported by backing the VM disk wrapper container with
a generic PrsistentVolumeClaim.  When the VM disk wrapper container starts up
to provide a VM disk to a new VM for the very first time, the image in the
container is written over to a PersistentVolumeClaim

Example: Expand a 1g qcow2 image in a wrapper container into a 100gb .raw image
on a persistent volume
```
qemu-img convert image.qcow2 /path/to/persistent/volume/mount/image.raw
```

From then on, where ever the VM is launched the wrapper container serves up the
persistent volume claim as the block device. 

**Live Migrations**

To take this a step further and support live migrations, we can decouple the
wrapper container from the virt-launcher pod and place the wrapper container
anywhere in the cluster. This will let multiple VMs reference the wrapper
container's block device during a migration at the same time and not tie the
life-cycle of the wrapper container to a specific VM.

# Why are we doing this again?

This proposal is a missing piece in merging VMs and containers into the same
user workflow. We are already working out the details required to connect both
pods and VMs into the same overlay network. With this proposal we'll be able to
converge pods and VMs into the same image repository.

Soon both the storage and network story will be the same for pods and VMs.

This proposal also aligns our usage of PersistentVolumeClaims with how they are
intended to be used. PersistentVolumeClaims are meant to be claimed by a pod.
By assigning PVC to a wrapper container's POD, we are using a supported
kubernetes use case.
