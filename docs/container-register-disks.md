# Storing VMI Disks in the Container Registry

## Motivation

KubeVirt is a drop in solution for running VMIs on an existing kubernetes
cluster. One of our goals in doing this is to leverage the tools and workflows
kubernetes has established for containers and apply them to virtualized
workloads.

For example, with KubeVirt VMIs are managed like pods. Users post VMIs to
kubernetes like pods and the VMIs will soon have the option to integrate into
the kubernetes overlay network like pods.

To continue down the path of allowing users to introduce VMI workloads into
an existing kubernetes cluster with as little friction as possible, the next
hurdle for us to overcome is tying VMI disk distribution into the same
mechanism used by pods. To do this, KubeVirt must have a way of storing and
accessing VMI disks using a container registry.

## Requirements
* Users must have a workflow for pushing VMI disks into a container registry.
* KubeVirt runtime must be able to consume VMI disks stored in the container
  registry and attach those disks to VMIs.

## Local Ephemeral Disk Use Cases

### Immutable VMIs booting from ephemeral disk

Users want to launch VMI workloads backed by local ephemeral storage. The VMI
workload does not need to remain persistent across VMI restarts and the workload
does not require live migration support.

### Immutable VMI Replication booting from ephemeral local disks.

Users want to horizontally scale VMI workloads backed by local ephemeral
storage. *Pending introduction and implementation of VMIGroup object*

## Ephemeral Non Bootable local disk storage

Users want to attach a non-bootable disk to a VMI preloaded with data. This data
could consist of anything. It could be configuration data, rpms, or anything
else a user might want to distribute with their VMIs.

## Implementation

### High Level Design

**Standardised KubeVirt VMI disk Wrapper**

KubeVirt provides a standardized base wrapper container image that serves up a
user provided VMI disk as a local file consumable by Libvirt. This base
container image does not have a VMI disk inside of it, it merely contains
everything necessary for serving up a VMI disk in a way Libvirt can consume.

**User Workflow**

Users push VMI Images into the container registry using the KubeVirt base
wrapper container image. By default, the base wrapper container KubeVirt
provides will serve up any VMI disk placed in the /disk directory as a block
device consumable by libvirt.

Example:
```
cat << END > Dockerfile
FROM kubevirt/container-disk-v1alpha
ADD fedora25.qcow2 /disk
END
docker build -t vmdisks/fedora25:latest .
docker push vmdisks/fedora25:latest
```
Users start a VMI using a "container registry backed VMI disk" by referencing the
image name they pushed to the registry. Any number of VMIs can be started
anywhere in the cluster referencing the same image.

Example:
```
kind: VirtualMachineInstance
spec:
  domain:
    devices:
      disks:
      - type: ContainerDisk:v1alpha
        - source:
          name: vmdisks/fedora25:latest
        - target:
          device: sda
```

**KubeVirt Runtime Implementation**

When virt-controller sees a VMI object with a disk type of
**ContainerDisk:v1alpha**, virt-controller places the image wrapper container
in the virt-launcher pod along with the container that monitors the VMI process.

When the virt-launcher pod starts, the wrapper container copies the user's VMI
image as file on a shared host directory.

When virt-handler sees a VMI is placed on the local node with the
**ContainerDisk:v1alpha** disk type defined, virt-handler injects the
necessary configuration into the domain xml required to add the disk backed by
a local file.

The **v1** part of the ContainerDisk disk type represents the standard
used during the virt-handler disk conversion process. As we gain more
experience with this feature, we may want to adopt a new standard for how VMI
images are wrapped by a container while maintaining backwards compatibility.
