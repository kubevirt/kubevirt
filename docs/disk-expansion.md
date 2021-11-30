# Disk expansion

## Enabling

For some storage methods, Kubernetes may support expanding storage in-use (allowVolumeExpansion feature).
KubeVirt can respond to it by making the additional storage available for the virtual machines.
This feature is currently off by default, and requires enabling a feature gate.
To enable it, add the ExpandDisks feature gate in the kubevirt object:

kubectl edit kubevirt -n kubevirt kubevirt
```yaml
spec:
  configuration:
    developerConfiguration:
      featureGates:
      - ExpandDisks
```

Enabling this feature does two things:
- Notify the virtual machine about size changes
- If the disk is a Filesystem PVC, the matching file is expanded to the remaining size (while reserving some space for file system overhead).

## Usage

To expand a disk, edit the matching PersistentVolumeClaim:

`kubectl edit pvc my-disk-pvc`

And increase the spec.resource.requests.storage to a larger size.
A running VMI will be notified that the disk has been expanded.
File systems remain unchanged - they need to be expanded to use the remaining data.

## Why do we not expand file systems?

An operating system may do its own caching of disk writes, and to expand a file
system we need to write to portions of the disk that are already in use. This
may result in corrupt data, unless the operating system expects this kind of
operation to happen.

For this reason we do not increase the file system size automatically.

## Why is the DataVolume size and the VirtualMachine size unchanged?

The DataVolume and VirtualMachine specs are currently immutable and are not updated to match the
growing PersistentVolumeClaim.

Additionally, DataVolumes are predecessors to PVC populators (still in progress), and in the future,
will be unlinked and garbage-collected by kubernetes once the import is done.  
They are not expected to continue to be used after the import is done.

If you wish to track the current PVC size for a given VirtualMachineInstance without finding the
matching PVC, you can inspect the vmi.status.volumeStatus PersistentVolumeClaimInfo field.
