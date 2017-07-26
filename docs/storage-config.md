# Virtual machine disk backend config

This document describes the approach to managing the backend configuration
for virtual machine disks, principally where the storage comes from and how it
is accessed & managed.

## Primary storage sources

There are a number of possible sources for virtual disk storage. QEMU is able
to consume a variety of storage backends, whether local files (in raw, qcow2 or
a number of other formats), local block devices (SCSI, iSCSI, RBD, etc), or
direct network block devices (iSCSI, RBD, GlusterFS, NBD, HTTP(s), etc).

The complication with dealing with local files is that, at time of writing, k8s
does not allow dynamically adding/removing volumes to running PODs. Since the
QEMU process is living in the libvirtd POD namespaces, any local storage has to
be made available at the time the libvirtd POD is started. Volumes that are made
available to the VM POD are not directly accessible from the libvirtd POD. There
may be some tricks that can be used to get access to VM POD storage, from the
libvirtd POD, but these would likely have to rely on internal impl details of
k8s which are not guaranteed / supported interfaces, so not desirable to do.

At this time there is no way to make host block devices directly accessible to
PODs. If this is enabled in the future, it will share the same problem as local
file storage, in that the libvirtd POD cannot access block devices associated
with the VM POD, so any block devices used would have to be devices that are
visible to the libvirtd POD when it is initially launched.

### Direct network block devices

Given the complications with local file/block storage, the simplest way to
configure a virtual disk is using a direct network block device. In the VM spec
this can be configured two ways.

#### Direct storage reference

The storage server can be referenced directly, for example, as an iSCSI server
address:

```yaml
    disks:
      - source:
          iscsi:
            host:
              name: iscsi-demo-target
              port: "3260"
            portal: iqn.2017-01.io.kubevirt:sn.42
	    lun: 2
```

This would also work for RBD, NBD, GlusterFS, and HTTP(s) protocols.

#### Indirect storage reference

The k8s PersistentVolumeClaim (PVC) resource is an abstract concept that allows
a tenant user to claim a PersistentVolume (PV). There are many different
backends for PVs, some are filesystem based and some are network block device
based. Using QEMU's direct network backend drivers, it is possible to directly
access the storage for any PV that is backend by a network block device (in
particular the iSCSI and RBD k8s backends).

```yaml
    disks:
      - source:
          pvc:
	    claimname: some-pvc-name
```

When launching QEMU the PVC resource is accessed to get the corresponding
PV resource. The configuration for the ISCSI / RBD backend is then extracted
to configure the QEMU storage backend


### Dynamic storage reference

Rather than having the storage volume pre-created upfront, it may be useful to
have a way to dynamically create storage backends. To achieve this, it is
possible to define a storage source which defines a container. The virt-handler
would create an instance of this container, which would be required to expose a
HTTP service with a well-known URI eg https://$POD-IP/volume

Making a PUT request to this URI would trigger creation of the volume, and a
a DELETE request would naturally delete it. The response to the PUT request
would be a YAML document following the 'iscsi' or 'pvc' schema for the VM
spec disk source. eg

```yaml
    disks:
      - source:
          container:
	    image: some-image-name
	    ... any other k8s container attribute...
```

Upon making the PUT request it would respond with

```yaml
    pvc:
      claimname: some-pvc-name
```

or

```yaml
    iscsi:
      host:
        name: iscsi-demo-target
        port: "3260"
      portal: iqn.2017-01.io.kubevirt:sn.42
    lun: 2
```

## Access modes

There is no general requirement that any disk source be used exclusively by a
single virtual machine. There are three permissible access modes

* Write-exclusive: a single virtual machine is writing to the storage. No
  other VM may access the storage whether for read or write.
* Write-shared: multiple virtual machines are permitted to access the storage,
  with concurrent writes permitted. This implies that the guest OS has some kind
  of cluster-aware filesystem over the disk, details of which are opaque to the
  host.
* Read-shared: multiple virtual machines are pmitted to access the storage, but
  all must be read-only. This allows arbitrary guest usage without needing any
  kind of cluster-aware filesystems.


## Copy mode

The default behaviour will be to directly access the referenced network storage
from a QEMU network driver. IOW any writes made by QEMU will be persistent
against the master storage.

It is, however, also valid to want local file / block device based storage for
the virtual machine, whose content is initialized by the master referenced
storage. This can be achieved by introducing the notion of a "copy mode" in the
disk backend configuration. This would accept the following options

* None: the storage is directly accessed by QEMU. This is the default.
* Copy-on-write: an overlay is provided with copy-on-write semantics against the
  master storage. The local overlay will only contain changes made by the guest.
* Copy-on-read: an overlay is provided with copy-on-read semantics against the
  master storage. The local overlay will contain both changes made by the guest,
  and copies of any blocks read.
* Clone: A full copy of the master storage is made to a local file. Once this is
  created, no further use is made of the master storage.

The first copy mode provides long term persistent storage, while the other modes
provide ephemeral storage that is discarded when the virtual machine is deleted
from the node.

The host needs to have somewhere to store the local overlays / files. This is
achieved by configuring one or more volumes for the libvirtd POD when it is
created, and mounting them into a defined location. Any valid PVC is usable for
this purpose. If the PVC backing store is only accessible for a single host,
then when performing migration, QEMU block storage migration feature will need
to be enabled. If the PVC references a networked filesystem that can be mounted
on multiple hosts, then VMs can migrate with no need to copy emphemeral disks.
If multiple volume mounts are made available to the libvirtd POD, annotations
would need to be used to tag them for different usage types. This might be useful
if a cluster needs to provide ephemeral storage with differing performance grades
or costs (eg one volume provides cheap & slow rotating rust, while another volume
provides expensive & fast solid state). The VM spec disk configuration would be
able to optionally include a reference to the usage type it desires, to allow
virt-handler to map the disk to the particular volume mount.

Thus the libvirtd container may contain

```yaml
    annotations:
      kubevirt.io/storage.usage-type.fast: /storage/ssd
      kubevirt.io/storage.usage-type.slow: /storage/rust
    volumeMounts:
      - mountPath: /storage/ssd
        name: storage-ssd
      - mountPath: /storage/rust
        name: storage-rust
```

NB The DaemonSet resource does not provide a way to dynamically assign
different PVCs to each libvirtd POD it creates. Thus in the short term, to make
use of this feature would require the cloud admin to manually create libvirtd
POD specs for each compute node, instead of using a DeamonSet resource. It is
anticipated that either DaemonSet would be enhanced eventually, or replaced by
a more advanced resource that can dynamically assigns PVCs to each POD.

When configuring the VM spec disk

```yaml
    disks:
      - source:
          pvc:
	    claimname: some-pvc-name
	access-mode: read-only
	copy:
	  mode: copy-on-write
          usage-type: fast
```

The 'usage-type' attribute refers to the 'kubevirt.io/sotrage.usage-type'
annotations provided by the libvirtd POD. Either all libvirtd PODs would need
to support the same annotations, or the scheduler will need to be able to only
place VMs on the nodes where the libvirtd POD has the neccessary annotations.

Each of the volume mounts provided for emphemeral storage would need to have a
libvirt storage pool associated with them. To create the ephemeral disks, the
virt-handler would thus use the storage pool APIs in libvirtd to create disks
whose backing store refers to the master PVC.


## Populating PVCs

The virtual machine startup code path should not care how the PVCs associated
with the disks are populated. It is assumed that the PVCs have been setup ahead
of time, either by the cloud administrator, or by the tenant user. There are a
number of possible approaches that could be used to populate these PVCs, the
choice of which is opaque to virt-handler, where merely needs the PVC name.

This doc will outline some high level concepts, but detailed technical designs
for them will be left for separate documents to define.  The PODs described
below could either be things that are run prior to VM launch, to statically
populate some PVCs with persistent data, or they could be used as part of the
dynamic disk source described above, to create storage on-demand during VM
startup.

### HTTP upload

A "HTTP upload" POD could be created that references the PVC that needs to
be populated. This POD would run a self-contained service that just exposes a
HTTP service that accepts a PUT request at a well known URL. Any data sent to
this URL would be immediately written into the PVC. Thus a tenant user can
populate an arbitrary PVC with data by simply launching a POD that references
the this 'kubevirt-pvc-upload' image, and a PVC, then use 'curl' to PUT the
payload.

### Container repository copy

A "contanier repository copy" POD could be created that references the PVC that
needs to be populated. This POD would run a self-contained service that just
pulls down a named image from a container repository, extracts a disk image file
(raw / qcow2 / whatever) from this image, and copies it into the PVC. When the
copy is complete the POD would exit


### Container repository stream

A "container repository stream" POD could be created that exposes an ISCSI server
with one or more LUNs. The POD would run a self-contained service that just
pulls down, one or more, named images from a container repository, extracting the
disk image files (raw / qcow2 / whatever) from each image. The extracted images
that then exported by the iSCSI server. The ISCSI LUNs can be directly
referenced by a VM spec disk image, or PVCs can be created to abtract away the
iSCSI server addresses.


### Container image to disk adapter

A "container image to disk adapter" POD could created that takes a docker
image, and auto-creates a set of stacked qcow2 files which are populated with
each layer in the contanier image. The POD would provide a kernel + initrd
image that would be booted, and then run the container image main entrypoint.
This essentially enables the "Clear Containers" usage scenario in Kubevirt by
allowing existing docker images to be directly consumed as if they were disk
images. Thus tenant users would never need to deal with qcow2 images directly,
they can use normal docker tools for building images & let kubevirt boot them
as-is.
