# Software Emulation

By default KubeVirt uses the `/dev/kvm` device to enable hardware emulation.
This is almost always desirable, however there are a few exceptions where this
approach is problematic. For instance, running KubeVirt on a cluster where the
nodes do not support hardware emulation.
In the same way, by default KubeVirt requires presence of `/dev/vhost-net`
in case that at least one network interface model is virtio (note: if the NIC
model is not explicitly specified, by default virtio is chosen).

If `useEmulation` is enabled,
- `qemu` will be used for software emulation, in case that hardware emulation
  via `/dev/kvm` is unavailable.
- QEMU userland virtio NIC emulation will be used for virtio-net interfaces,
  in case that in-kernel virtio-net backend emulation via `/dev/vhost-net` 
  is unavailable.

If `useEmulation` is disabled, and a required hardware emulation device is unavailable
(`/dev/kvm`, or `/dev/vhost-net` for a VirtualMachine which uses virtio for at least one interface),
the VirtualMachine will fail to start and an error will be reported.

Note that software emulation, when enabled, is only used as a fallback when
hardware emulation is not available. Hardware emulation is always attempted first,
regardless of the value of the `useEmulation`.

# Configuration

Enabling software emulation is a cluster-wide setting, and is activated by
editing the `KubeVirt` CR as follows:

```bash
cluster-up/kubectl.sh --namespace kubevirt edit kubevirt kubevirt
```

Add the following snippet to the spec:

```yaml
spec:
  configuration:
    developerConfiguration:
      useEmulation: true
```
