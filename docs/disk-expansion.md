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
