# Software Emulation

By default KubeVirt uses the `/dev/kvm` device to enable hardware emulation.
This is almost always desirable, however there are a few exceptions where this
approach is problematic. For instance, running KubeVirt on a cluster where the
nodes do not support hardware emulation.

If `allowEmulation` is enabled, hardware emulation via `/dev/kvm` will still be
attempted first. Software emulation will only be used if the device is
unavailable.

# Configuration

Enabling software emulation as a fallback is a cluster-wide setting, and is
activated using a ConfigMap in the `kube-system` namespace. It can be enabled
with the following command:

```bash
cluster/kubectl.sh --namespace kube-system create configmap kubevirt-config \
    --from-literal debug.allowEmulation=true
```

If the `kube-system/kubevirt-config` ConfigMap already exists, the above entry
can be added using:


```bash
cluster/kubectl.sh --namespace kube-system edit configmap kubevirt-config
```

In this case, add the `debug.allowEmulation: "true"` setting to `data`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubevirt-config
  namespace: kube-system
data:
  debug.allowEmulation: "true"

```

**NOTE**: The values of `ConfigMap.data` are **strings**. Yaml requires the use of
quotes around `"true"` to distinguish the value from a boolean.
