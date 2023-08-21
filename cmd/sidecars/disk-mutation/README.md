# KubeVirt Root disk mutation hook sidecar

To use this hook, use following annotations:

```yaml
annotations:
  # Request the hook sidecar
  hooks.kubevirt.io/hookSidecars: '[{"image": "registry:5000/kubevirt/example-disk-mutation-hook-sidecar:devel"}]'
  # Overwrite the disk image name
  diskimage.vm.kubevirt.io/bootImageName: "virt-disk.img"
```
