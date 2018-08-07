# Using Persistent Volumes as Virtual Machine disks

Author: Fabian Deutsch \<fabiand@redhat.com\>

## Introduction

Virtual Machines are able to have disks attached. They are not always required,
but have some value if you need to persist data.

Kubernetes provides persistent storage through Persistent Volumes and Claims.

The purpose of this document is to describe the mechanism to use Persistent
Volumes as a backing store for Virtual Machine disks.


### Use-case

The primary use-case is to attach regular (writable) disks to Virtual Machines
which are backed by Persistent Volumes.


## API

This section describes how Persistent Volumes are referenced in the `VMI`
Resource type.

In general the referencing is aligned with how pods are consuming Persistent
Volume Claims as described [here](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#persistentvolumeclaim-v1-core)

Today the `VMI.spec.domain` reflects much of [libvirt's domain xml specification](http://libvirt.org/formatdomain.html#elementsDisks). To
communicate the new storage type through the API, an additional disk type
`PersistentVolumeClaim` is accepted. In the case of a `PersistentVolumeClaim`
type The `disk/source/name` attribute is used to name the claim to use.

Example with the following PV and PVC:

```yaml
# For the sake of completeness a volume and claim
kind: PersistentVolume
metadata:
  name: pv001
  labels:
    release: "stable"
spec:
  capacity:
    storage: 5Gi
  iscsi:
    targetPortal: example.com:3260
    iqn: iqn.2013-07.com.example:iscsi-nopool/
    lun: 0
---
kind: PersistentVolumeClaim
metadata:
  name: disk-01
spec:
  resource:
    requests:
      storage: 4Gi
  selector:
    matchLabels:
release: "stable"
```

this is used by the VMI in the following way:

```yaml
kind: VirtualMachineInstance
spec:
  domain:
    devices:
      disks:
      - type: PersistentVolumeClaim
        source:
          name: disk-01
        target:
          bus: scsi
          device: sda
```


## Implementation

### Flow

1. User adds an existing `PersistentVolumeClaim` as described above to the VMI
instance.
2. The VMI Pod is getting scheduled on a host, the `virt-handler`
   identifies the Claim, translates it into the corresponding
   libvirt representation and includes it in the domain xml.

**Note**: The `virt-controller` does not do anything with the claim.

**Note**: In this flow, the claim is _not_ used by the pod, the claim is only
used by the `virt-handler` to identify the connection details to the storage.

Because VMIs only accept block storage as disks, the handler can only accept
claims which are backed by block storage types. Currently only iSCSI support is
implemented.


### `virt-handler` behavior

Once a VMI is scheduled on a host, the `virt-handler` will transform the VMI Spec
into a libvirt domain xml.

During this transformation, every disk which is of type `PersistentVolumeClaim`
will be converted into an adequate/equivalent libvirt disk type.

To achieve this, the `virt-handler` needs to look at volume attached to a
claim, to identify the `type` and connection details. Note that the claim
itself will contain the connection details if it is associated with a volume.
In the case that the claim is not associated with a volume the launch of the VMI
will fail. If the type is iSCSI, then it can take the connection details and
transform them into the correct libvirtd representation.

Example for an iSCSI volume:

```xml
  <disk type='network' device='disk'>
    <driver name='qemu' type='raw'/>
    <source protocol='iscsi' name='iqn.2013-07.com.example:iscsi-nopool/2'>
      <host name='example.com' port='3260'/>
    </source>
  </disk>
```

In the snippet above, most of the snippet is hard-coded, only the values for
`disk/source/name`, `disk/host/name`, and `disk/host/port` are populated with
the values from the Persistent Volume Claim.

Once the VMI starts up, `qemu` will be connecting to the target to connect the
disk.
