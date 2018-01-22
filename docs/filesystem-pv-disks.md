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
only iSCSI) to a VM.

The mechanism proposed in this proposal is leveraging Kubernetes to attach
the remote storage as a file-system to a container, to use this file-system to
store a disk image to act as a backend storage to the VM's disk.

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
      - name: root-disk
        volumeName: my-fs-store
  volumes:
  - name: my-fs-store
    persistentVolumeClaim: my-fs-claim
```

Here the user attaches the PersistentVolumeClaim `my-fs-claim` as a disk to a
VM.


### Storage Type Inference

The system needs to know if the referenced volume needs to be treated as a
block or file-system volume.
This information can be infered from the existing PV metadata.


## Implementation

### Volume layout

A file-system based volume will contain only a single image file, this file
must be named `disk.img`.
The format of the file must be `raw`.

The file-system layout of a mounted volume then looks like:

```
/disk.img
```

In future we might want to add additional files to carry metadata, but the limit
of a single image file per volume must not be changed.


## Additional Notes

### Introduction of a `driver` field

In future we might want to introduce a `driver` field for disks to
differentiate between (up to now) qemu's built-in drivers or using kubelet's
file-system and ([in close future](https://github.com/kubernetes/community/pull/805))
raw block storage support, i.e.:
```yaml
      - type: PersistentVolumeClaim
        driver:
          name: qemu-built-in
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
