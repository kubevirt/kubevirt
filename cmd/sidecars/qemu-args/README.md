# KubeVirt SMBIOS hook sidecar

To use this hook, use following annotations:

```yaml
annotations:
  # Request the hook sidecar
  hooks.kubevirt.io/hookSidecars: '[{"args": ["--version", "v1alpha2"], "image":"registry/example-example-qemu-args-sidecar:devel"}]'
  # Append additional qemu args
  qemu.vm.kubevirt.io/additionalArgs: '-fw_cfg name=opt/ovmf/X-PciMmio64Mb,string=65535'
```

## To build
To build sidecar image run the following from project root:

```shell
DOCKER_PREFIX=registry DOCKER_TAG=devel PUSH_TARGETS=example-qemu-args-sidecar make push
```

## To verify
Based on value for annotation key `qemu.vm.kubevirt.io/additionalArgs` the additional args will be added to the domain. 

This can be verified by running the following and checking the qemu commandLine section

```shell
kubectl exec -it virt-launcher-pod-name -- virsh dumpxml domainName
```
