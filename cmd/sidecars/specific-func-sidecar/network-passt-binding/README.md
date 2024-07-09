# KubeVirt Network Passt Binding Plugin

## Summary

Passt network binding plugin configures VMs Passt interface using Kubevirts hook sidecar interface.

It will be used by Kubevirt to offload Passt networking configuration.

> _NOTE_:
> Passt network binding is supported for pod network interfaces only.

# How to use

Register the `passt` binding plugin with its sidecar image:

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
        passt:
          sidecarImage: registry:5000/kubevirt/network-passt-binding:devel
  ...
```

In the VM spec, set interface to use `passt` binding plugin:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: vmi-passt
spec:
  domain:
    devices:
      interfaces:
      - name: passt
        binding:
          name: passt
  ...
  networks:
  - name: passt-net
    pod: {}
  ...
```
