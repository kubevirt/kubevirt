# Software Emulation

By default KubeVirt uses the `/dev/kvm` device to enable hardware emulation.
This is almost always desirable, however there are a few exceptions where this
approach is problematic. For instance, running KubeVirt on a cluster where the
nodes do not support hardware emulation.
In the same way, by default KubeVirt requires presence of `/dev/vhost-net`
in case that at least one network interface model is virtio (note: if the NIC
model is not explicitly specified, by default virtio is chosen).

If `useEmulation` is enabled,
- hardware emulation via `/dev/kvm` will not be attempted. `qemu` will be used
  for software emulation instead.
- in-kernel virtio-net backend emulation via `/dev/vhost-net` will not be
  attempted. QEMU userland virtio NIC emulation will be used for virtio-net
  interface instead.

# Configuration

Enabling software emulation is a cluster-wide setting, and is activated by
editing the kubevirt-config as follows:

```bash
cluster-up/kubectl.sh --namespace kubevirt edit kubevirt kubevirt
```

Add the following snippet to the spec:

```yaml
spec:
  developerConfiguration:
    useEmulation: "true"
```

**NOTE**: The values of `KubeVirt.spec` are **strings**. Yaml requires the use of
quotes around `"true"` to distinguish the value from a boolean.
