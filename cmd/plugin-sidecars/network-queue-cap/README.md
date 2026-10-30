# Network queue cap plugin

Lowers virtio `driver.queues` on the generated domain when a VM sets `network.kubevirt.io/max-queues`. This is a [VEP 190](https://github.com/kubevirt/enhancements/issues/190) launcher sidecar hook, not a legacy `hooks.kubevirt.io/hookSidecars` container.

The `Plugins` feature gate must be enabled. The value is an integer from 1 to 256. A missing or invalid value leaves the domain unchanged. Non-virtio interfaces are left unchanged. The VM keeps its existing masquerade or bridge interfaces.

The socket path is `/var/run/kubevirt-plugin/network-queue-cap/hook.sock`. A MutatingAdmissionPolicy injects this image into virt-launcher and mounts that directory from the `kubevirt-plugin-sockets` volume, the same way as `cmd/plugin-sidecars/test-launcher-hook`. The hook point is `GuestDefinition`.

```yaml
apiVersion: plugin.kubevirt.io/v1alpha1
kind: Plugin
metadata:
  name: network-queue-cap
spec:
  condition: '"network.kubevirt.io/max-queues" in vmi.Annotations'
  launcherHooks:
    - sidecar:
        socketPath: /var/run/kubevirt-plugin/network-queue-cap/hook.sock
        permittedHooks:
          - GuestDefinition
  mutatingAdmissionPolicies:
    - name: network-queue-cap
```

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: windows-multi-nic
spec:
  template:
    metadata:
      annotations:
        network.kubevirt.io/max-queues: "2"
    spec:
      domain:
        devices:
          networkInterfaceMultiqueue: true
          interfaces:
            - name: default
              masquerade: {}
      networks:
        - name: default
          pod: {}
```
