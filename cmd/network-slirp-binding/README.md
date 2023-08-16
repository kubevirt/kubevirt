# KubeVirt Network Slirp Binding Plugin

## Summary

Slirp network binding plugin configures VMs Slirp interface using Kubevirts hook sidecar interface.

It will be used by Kubevirt to offload slirp networking configuration.

> _NOTE_:
> Slirp network binding is supported for pod network interfaces only.

# How to use

Register the `slirp` binding plugin with its sidecar image:

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
        slirp:
          sidecarImage: registry:5000/kubevirt/network-slirp-binding:devel
  ...
```

In the VM spec, set interface to use `slirp` binding plugin:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: vmi-slirp
spec:
  domain:
    devices:
      interfaces:
      - name: slirp
        binding:
          name: slirp
  ...
  networks:
  - name: slirp-net
    pod: {}
  ...
```
