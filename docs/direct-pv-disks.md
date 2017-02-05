# Using Persistent Volumes as Virtual Machine disks

Author: Fabian Deutsch \<fabiand@redhat.com\>

## Introduction

Virtual Machines use to have disks attached. They are not always required, but
have some value if you need to persist data.

Kubernetes provides persistent storage through Persistent Volumes and
Claims.

The purpose of this proposal is to describe a mechanism to use Persistent
Volumes as a backing store for Virtual Machine disks.


### Use-case

The primary use-case is to attach regular (writable) disks to Virtual Machines
which are backed by Peristent Volumes.


## API

This section is concerned about how the Persistent Volumes are referenced
in the `VM` Resource type.

In general the referencing is aligned with how pods are consuming Persistent
Volume Claims as described [here](https://kubernetes.io/docs/api-reference/v1.5/#persistentvolumeclaimvolumesource-v1)

Today the `VM.spec.domain` reflects much of
[libvirt's domain xml specification](http://libvirt.org/formatdomain.html#elementsDisks).
To communicate the new storage type through the API, an additional disk type
`PersistentVolumeClaim` is accepted.
In the case of a `PersistentVolumeClaim` type The `disk/source/name` attribute is
used to name the claim to use.

Example with the following PV and PVC:

```yaml
# For teh sake of completeness a volumen and claim
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

this is used by the Vm in the following way:

```yaml
kind: VM
spec:
  domain:
    devices:
      disks:
      - type: PersistentVolumeClaim
        - source:
          name: disk-01
        - target:
          bus: scsi
          target: sda
```


## Implementation

### Flow

1. User adds an existing `PersistentVolumeClaim` as described above to
   the VM instance.
2. The VM Pod is getting scheduled on a host, the `virt-handler`
   identifies the Claim, translates it into the corresponding
   libvirt representation and includes it in the domain xml.

**Note**: The `virt-controller` does not do anything with the claim.

**Note**: In this flow, the claim is _not_ used by the pod, the claim
is only used by the `virt-handler` to identify the connection details
to the storage.

Because VMs only accept block storage as disks, the handler can only
accept claims which are backed by block storage types.


### `virt-handler` changes

Once a VM is scheduled on a host, the `virt-handler` is transforming the
VM Spec into a libvirt domain xml.

During this transformation, every disk which is of type `PersistentVolumeClaim`
needs to be transformed into an adequate libvirt disk type.

To achieve this, the `virt-handler` needs to look at volume attached to a
claim, to identify the `type` and connection details.
Note that the claim itself will contain the connection details if it is
associated with a volume. In the case that the claim is _not_ associated with
a volume the launch of the VM should fail.
If the type is either iSCSI or RBD, then it can take the connection details
and transform them into the correct libvirtd representation.

Example for an iSCSI volume:

```xml
  <disk type='network' device='disk'>
    <driver name='qemu' type='raw'/>
    <source protocol='iscsi' name='iqn.2013-07.com.example:iscsi-nopool/2'>
      <host name='example.com' port='3260'/>
    </source>
  </disk>
```

In the snippet above, most of the snippet is hard-coded, only the values
for `disk/source/name`, `disk/host/name`, and `disk/host/port` are populated
with the values from the Persistent Volume Claim.

Once the VM starts up, `qemu` will be connecting to the target to connect
the disk.
