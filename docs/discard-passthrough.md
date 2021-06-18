# Thick and thin volume provisioning

Sparsification can make a disk thin-provisioned, in other words it allows to convert the freed space within the disk image into free space back on the host. The [fstrim](https://man7.org/linux/man-pages/man8/fstrim.8.html#:~:text=fstrim%20is%20used%20on%20a,unused%20blocks%20in%20the%20filesystem) utility can be used on a mounted filesystem to discard the blocks not used by the filesystem. In order to be able to sparsify a disk inside the guest, the disk needs to be configured in the [libvirt xml](https://libvirt.org/formatdomain.html) with the option `discard=unmap`. In KubeVirt, every disk is passed as default with this option enabled. It is possible to check if the trim configuration is supported in the guest by running`lsblk -D`, and check the discard options supported on every disk. Example:
```bash
$ lsblk -D
NAME   DISC-ALN DISC-GRAN DISC-MAX DISC-ZERO
loop0         0        4K       4G         0
loop1         0       64K       4M         0
sr0           0        0B       0B         0
rbd0          0       64K       4M         0
vda         512      512B       2G         0
└─vda1        0      512B       2G         0
```

However, in certain cases like preallocaton or when the disk is thick provisioned, the option needs to be disabled. The disk's PVC has to be marked with an annotation that contains `/storage.preallocation` or `/storage.thick-provisioned`, and set to true. If the volume is preprovisioned using [CDI](https://github.com/kubevirt/containerized-data-importer) and the [preallocation](https://github.com/kubevirt/containerized-data-importer/blob/main/doc/preallocation.md) is enabled, then the PVC is automatically annotated with: `cdi.kubevirt.io/storage.preallocation: true` and the discard passthrough option is disabled.

Example of a PVC definition with the annotation to disable discard passthrough:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc
  annotations:
    user.custom.annotation/storage.thick-provisioned: "true"
spec:
  storageClassName: local
  accessModes:
    - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 1Gi
```

