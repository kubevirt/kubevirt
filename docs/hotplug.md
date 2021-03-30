# Hotplug Volumes

KubeVirt now supports hotplugging volumes into a running Virtual Machine Instance (VMI). The volume must be either a block volume or contain a disk image file just like any other regular volume.

## Enable feature gate

In order to enable the HotplugVolumes feature gate, you must add the HotplugVolumes to the list of enabled featureGates.

```yaml
spec:
  configuration:
    developerConfiguration:
      featureGates:
      - HotplugVolumes
```

Once the feature gate is enabled, you will be able to hotplug volumes into a running VMI.

## Virtctl support

In order to hotplug a volume, you must first prepare a volume. This can be done by using a DataVolume (DV). In the example we will use a blank DV in order to add some extra storage to a running VMI

```yaml
apiVersion: cdi.kubevirt.io/v1beta1
kind: DataVolume
metadata:
  name: example-volume-hotplug
spec:
  source:
    blank: {}
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
```
In this example we are using ReadWriteOnce accessMode, and the default FileSystem volume mode. Volume hotplugging supports all combinations of block volume mode and ReadWriteMany/ReadWriteOnce/ReadOnlyMany accessModes, if your storage supports the combination.

### Addvolume

Now lets assume we have started a VMI like the [Fedora VMI in examples](examples/vmi-fedora.yaml) and the name of the VMI is 'vmi-fedora' we can add the above blank volume to this running VMI by using the 'addvolume' command  available with virtctl

```bash
$ virtctl addvolume vmi-fedora --volume-name=example-volume-hotplug
```

This will hotplug the volume into the running VMI, and set the serial of the disk to the volume name. In this example it is set to example-hotplug-volume.

#### Serial
You can change the serial of the disk by specifying the --serial parameter, for example:
```bash
$ virtctl addvolume vmi-fedora --volume-name=example-volume-hotplug --serial=1234567890
```

The serial will be used in the guest so you can identify the disk inside the by the serial. For instance in Fedora the disk by id will contain the serial
```bash
$ virtctl console vmi-fedora

Fedora 32 (Cloud Edition)
Kernel 5.6.6-300.fc32.x86_64 on an x86_64 (ttyS0)

SSH host key: SHA256:c8ik1A9F4E7AxVrd6eE3vMNOcMcp6qBxsf8K30oC/C8 (ECDSA)
SSH host key: SHA256:fOAKptNAH2NWGo2XhkaEtFHvOMfypv2t6KIPANev090 (ED25519)
eth0: 10.244.196.144 fe80::d8b7:51ff:fec4:7099
vmi-fedora login:fedora
Password:fedora
[fedora@vmi-fedora ~]$ ls /dev/disk/by-id
scsi-0QEMU_QEMU_HARDDISK_1234567890
[fedora@vmi-fedora ~]$ 
```
As you can see the serial is part of the disk name, so you can uniquely identify it.

### Removevolume
In addition to hotplug plugging the volume, you can also unplug it by using the 'removevolume' command available with virtctl
```bash
$ virtctl removevolume vmi-fedora --volume-name=example-volume-hotplug
```
Note: You can only unplug volumes that were dynamically added with addvolume, or using the API.

### VolumeStatus
VMI objects have a new VolumeStatus status field. This is an array containing each disk, hotplugged or not. For example after hotplugging the volume in the addvolume example, the VMI status will contain this:
```yaml
    volumeStatus:
    - name: cloudinitdisk
      target: vdb
    - name: containerdisk
      target: vda
    - hotplugVolume:
        attachPodName: hp-volume-7fmz4
        attachPodUID: 62a7f6bf-474c-4e25-8db5-1db9725f0ed2
      message: Successfully attach hotplugged volume volume-hotplug to VM
      name: example-volume-hotplug
      phase: Ready
      reason: VolumeReady
      target: sda
```
Vda is the container disk that contains the Fedora OS, vdb is the cloudinit disk. As you can see those just contain the name and target used when assigning them to the VM. The target is the value passed to QEMU when specifying the disks. The value is unique for the VM and does *NOT* represent the naming inside the guest. For instance for a Windows Guest OS the target has no meaning. The same will be true for hot plugged volumes. The target is just a unique identifier meant for QEMU, inside the guest the disk can be assigned a different name.

The hotplugVolume has some extra information that regular volume statuses do not have. The attachPodName is the name of the pod that was used to attach the volume to the node the VMI is running on. If this pod is deleted it will also stop the VMI as we cannot guarantee the volume will remain attached to the node. The other fields are similar to conditions and indicate the status of the hot plug process. Once a Volume is ready it can be used by the VM.

Note: Currently every volume hotplugged requires an additional pod to be created.

## Live Migration
Currently Live Migration is disabled for any VMI that has volumes hotplugged into it. This limitation will be removed in a future release.