# Software Emulation

By default KubeVirt uses the `/dev/kvm` device to enable hardware emulation.
This is almost always desirable, however there are a few exceptions where this
approach is problematic. For instance, running KubeVirt on a cluster where the
nodes do not support hardware emulation.

If `useEmulation` is enabled, hardware emulation via `/dev/kvm` will not be
attempted. `qemu` will be used for software emulation instead.

# Configuration

Enabling software emulation is a cluster-wide setting, and is activated using a
ConfigMap in the `kube-system` namespace. It can be enabled with the following
command:

```bash
cluster/kubectl.sh --namespace kube-system create configmap kubevirt-config \
    --from-literal debug.useEmulation=true
```

If the `kube-system/kubevirt-config` ConfigMap already exists, the above entry
can be added using:

```bash
cluster/kubectl.sh --namespace kube-system edit configmap kubevirt-config
```

In this case, add the `debug.useEmulation: "true"` setting to `data`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubevirt-config
  namespace: kube-system
data:
  debug.useEmulation: "true"

```

**NOTE**: The values of `ConfigMap.data` are **strings**. Yaml requires the use of
quotes around `"true"` to distinguish the value from a boolean.
