# KubeVirt rbd_hotplug hook sidecar

To use this hook, use following annotations:

```yaml
annotations:
  # Request the hook sidecar
  hooks.kubevirt.io/hookSidecars: '[{"image": "registry:5000/kubevirt/rbd-hotplug-sidecar:devel"}]'
  # Overwrite base board manufacturer name
  rbd-hotplug.vm.kubevirt.io/user: admin
  rbd-hotplug.vm.kubevirt.io/secret: "XXXXX"
  rbd-hotplug.vm.kubevirt.io/monitors: '[{"host": "10.0.0.1"},]'
  rbd-hotplug.vm.kubevirt.io/attachments: '[{"pool": "test", "volume": "pvc-328a92d056cd11e8", "device": "vdf", "bus": "virtio"}]'
```