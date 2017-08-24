# Using file-system based PersistentVolumes as virtual disk backends

## Introduction

So far Kubernetes PersistentVolumes are remote file-systems attached to a pod at
runtime. This is currently the only way of exposing remote or local storage to
pods, through a file-system.
Despite the fact that storage is exposed as file-systems to pod, KubeVirt has
no mechanism to use these plain file-system based PVs to act as backends for
virtual disks.


## Goals

This proposal is about adding a simple mechanism to allow using file-system
based PersistentVolumes for storing file images backing virtual disks.

The focus is primarily on providing the right cluster levels mechanics to enable
future changes for improving usability, performance, or other characteristics.


## Non-Goals

- Provide a totally automatic solution
- Cover snapshots explicitly


## Use-case

The primary use-case is to allow a VM to use PVs as backend storage for their
virtual disks.


## Design Overview

Currently KubeVirt only supports [direct PV disks](direct-pv-disks.md).
This feature works, by assigning PVCs backed by network block storage (currently
only iSCSI) to a VM. The connection details of the PVC are then leveraged with
qemu's built-in network storage drivers, to directly connect qemu to the remote
storage. This effectively bypasses all other Kubernetes or KubeVirt components,
and establishes a direct connection between qemu and the storage backend.

The mechanism proposed in proposal however is leveraging Kubernetes to attach
the remote storage as a file-system to a pod, to use this file-system to storage
disk images which are acting as a backend storage to the VM.

To keep the story compatability between the block and file-system storage, this
proposal assumes that the file-system based storage will also just support a 1:1
mapping between the virtual disk and volume.
This aligns with the fact that a block volume can also just back a single
virtual disk.


## API

The general API to use PersistentVolume claims as virtual disk backends was
introduced with [direct PV proposal](direct-pv-disks.md).

The change suggested by this proposal, does not require any API change.

To use a PV as a virtual disk backend, a user needs to create a claim for the
required PV, this claim is then used as a disk source for a virtual disk.

An example:

```yaml
kind: VM
spec:
  domain:
    devices:
      disks:
      - type: PersistentVolumeClaim
        source:
          name: vm-01-disk
        target:
          bus: scsi
          device: sda
```

Here the user attaches the PersistentVolumeClaim _vm-01-disk_ to a VM, the
assumption is that the _vm-01-disk_ volume is a file-system based volume.


### Storage Type Inference

The system needs to know if the referenced volume needs to be treated as a
block or file-system volume. This information can be infered from the existing
PV metadata.


## Implementation

### Volume layout

A file-system based volume will contain the image file only, this file must be
named `disk.img`.
The format of the file must be `raw`.

The file-system layout of a mounted volume then looks like:

```
/disk.img
```

### Mounting & sharing

The file-system volume needs to be associated with the launcher container of
the VM pod, and is thus mounted in the VMs pod launcher mount namespace.

As the qemu proccesses run in the libvirt's mount namespace, the volume mount
namespace has to be shared with libvirt.

To achieve this, libvrit needs to gain access to the `/var/lib/kubelet/pods`
path in the `kubelet`'s mount namespace.
This path contains all volume mounts of all containers.

The handler can craft the (relative) path to a volume, by taking the pod's UUID
and volume informations. This path is then passed to libvirt, which in turn uses
it in a disk definition.

FIXME define the EXACT way of how to craft the path. 


## Additional Notes

### Introduction of a `driver` field

In future we might want to introduce a `driver` field for disks to
differentiate between (up to now) qemu's built-in drivers or using kubelet's
file-system and ([in close future](https://github.com/kubernetes/community/pull/805))
raw block storage support, i.e.:
```yaml
      - type: PersistentVolumeClaim
        driver:
          name: qemu
        source:
          name: vm-01-disks
---
      - type: PersistentVolumeClaim
        driver:
          name: kubelet
        source:
          name: vm-01-disks
```

### Snapshots

Also something for a different proposal, but to be considered are snapshots.
A general approach to snapshots which works with the existing direct PV and
this proposal is, to either use intermediate transparent qcow2 files, or improve
qemu to have a cow subsystem, which is agnostic to the backing store type.
Both solutions however would be independent of the storage type and don't
contradict with our designs.

### Virtfs

Virtfs might allow us to directly use file-systems as a backing store for
virtual machines.
This API design should not contradict with this use-case, but it will probably
depend on the `driver` field mentioned above to signal the system how the PV has
to beconsumed (mounted).

### `VMConfig` level feature based usage

It might be of value to users to only specify a PV, and not care about the file
to be used for a disk. We can add such logic, but this should probably reside on
the `VMConcifg` level. The `VM` API should focus to provide the mechanics to
support this more opinionated approach.
