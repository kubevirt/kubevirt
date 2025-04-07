# KubeVirt Network Bridge Binding Plugin

## Summary

Bridge network binding plugin configures VMs Bridge interface using Kubevirts hook sidecar interface. It also runs
his own DHCPd server in order to ensure consistency of the network configuration when LiveMigration is processed.

It will be used by Kubevirt to offload Bridge networking configuration. 

> _NOTE_:
> Bridge network binding is supported for pod network interfaces only.

# How to use

Register the `bridge` binding plugin with its sidecar image:

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    network:
      binding:
        bridge:
          sidecarImage: registry:5000/kubevirt/network-bridge-binding:devel
  ...
```

In the VM spec, set interface to use `bridge` binding plugin:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: vmi-bridge
spec:
  domain:
    devices:
      interfaces:
      - name: default
        binding:
          name: bridge
  ...
  networks:
  - name: default
    pod: {}
  ...
```
