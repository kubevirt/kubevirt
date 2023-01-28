# KubeVirt QEMU command line arguments hook sidecar

To use this hook, first be sure that `Sidecar` feature gate is enabled. You can check how to enable
it [here][]

To use this hook, use following annotations:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  annotations:
    hooks.kubevirt.io/hookSidecars: '[{"args": ["--version", "v1alpha2"], "image":
      "registry:5000/kubevirt/debug-toolkit:devel"}]'
    libvirt.vm.kubevirt.io/qemuArgs: '{"env": [{"name": "G_MESSAGES_DEBUG", "value": "all"}],
      "arg": [{"value": "-name"}, {"value": "guest=sidecarKubevirtVM,debug-threads=on"}]}'
```

This sidecar uses libvirt's interface for [QEMU command-line passthrough][].

## Example

```shell
# Create a VM requesting the hook sidecar
cluster-up/kubectl.sh create -f examples/vmi-sidecar-qemu-args.yml

# Once the VM is ready, connect to its display and login using name and password "fedora"
# Once the VM is ready, you can check if VM's domain has your custom configuration by
# running virsh dump.

# First get the virt-launcher's name
cluster-up/kubectl.sh get pods
NAME                                        READY   STATUS    RESTARTS   AGE
local-volume-provisioner-ln6vh              1/1     Running   0          49m
virt-launcher-vmi-sidecar-qemu-args-6fxsq   3/3     Running   0          3m50s

cluster-up/kubectl.sh exec --stdin --tty virt-launcher-vmi-sidecar-qemu-args-6fxsq -- \
  /usr/bin/virsh dumpxml default_vmi-sidecar-qemu-args

...
  <qemu:commandline>
    <qemu:arg value='-name'/>
    <qemu:arg value='guest=sidecarKubevirtVM,debug-threads=on'/>
    <qemu:env name='G_MESSAGES_DEBUG' value='all'/>
  </qemu:commandline>
...
```

# GDB to QEMU

QEMU supports listen for an incoming connection from gdb. By adding `-s` to QEMU command line, it'll
expect gdb connection to TCP port `1234`.

## Example

After enabling `featureGates` for `Sidecar`, use the following annotation to attach `-s` to QEMU
command line:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  annotations:
    hooks.kubevirt.io/hookSidecars: '[{"args": ["--version", "v1alpha2"], "image":
      "registry:5000/kubevirt/debug-toolkit:devel"}]'
    libvirt.vm.kubevirt.io/qemuArgs: '{"arg": [{"value": "-s"}]}'
```

In the context of `make cluster-sync` enviroment, you will need to port-foward

```shell
./cluster-up/kubectl.sh port-forward my-virt-launcher-pod 1234:1234
```

If you want to configure your own port or make QEMU wait till gdb is connected, see further
instructions in [QEMU's gdb usage][] documentation.

[here]: https://kubevirt.io/user-guide/operations/activating_feature_gates/
[QEMU command-line passthrough]: https://libvirt.org/kbase/qemu-passthrough-security.html
[QEMU's gdb usage]: https://qemu-project.gitlab.io/qemu/system/gdb.html
