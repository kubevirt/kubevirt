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

- Provide a totally atumatic solution
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


## API

The general API to use PersistentVolume claims as virtual disk backends was
introduced with [direct PV proposal](direct-pv-disks.md).

This proposal is merely adding an additional field to support addressing files
on a file-system.

Thus, to use a PV as a virtual disk backend, a user needs to create a claim for
the required PV, and then reference the to be used disk image using the newly
introduced `file` parameter.
The `file` field takes a path, relative to the source of the file-system held by
the referenced PVC.

An example:

```yaml
kind: VM
spec:
  domain:
    devices:
      disks:
      - type: PersistentVolumeClaim
        source:
          name: vm-01-disks
          file: disk-01.img  # The change
        target:
          bus: scsi
          device: sda
```

Here the user attaches the PersistentVolumeClaim _vm-01-disks_ to a VM, and uses
the file `disk-01.img` as the backing file for the disk `sda`.

### Storage Type Inference

The system can look at the PVC to infer whether file-system or raw block storage
should be used with a PV.
If there is a conflict between the VM API configuration and the backing PV, then
an error must be raised.
One error condition for example would be if a `file` field is given, but the
backing PV is of `volumeType: block`.


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
