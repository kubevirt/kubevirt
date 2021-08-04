# SSH into a VirtualMachineInstance

Every VM and VMI provides a `/portforward` subresource that can be used to create a websocket backed
network tunnel to a port inside the instance similar to Kubernetes pods.

One use-case for this subresource is to forward SSH traffic into the VMI either from the CLI
or a web-UI.

## Usage

### virtctl

To connect to a VMI from your local machine, `virtctl` provides the `ssh` command as a lightweight
SSH client. Refer to the commands help for more details.

```sh
virtctl ssh testvmi
```

#### Port Forward

If you prefer to use your local OpenSSH client, the `virtctl port-forward` command provides an option
to tunnel a single port to your local Stdout/Stdin.
This allows for the command to be used in the `ProxyCommand` option.

```sh
ssh -o 'ProxyCommand=virtctl port-forward --stdio=true testvmi 22' user@testvmi.mynamespace
```

This can also be used with `scp`.

To provide easier access to different VMs you can add the following to your `ssh-config`:

```
Host vmi/*
   ProxyCommand virtctl port-forward --stdio %h %p
Host vm/*
   ProxyCommand virtctl port-forward --stdio %h %p
```

This allows you to simply call `ssh fedora@vmi/testvmi.default` and your config and virtctl will do the rest.
Using this setup it also becomes trivial to setup different identities for different namespaces inside your `ssh-config` 

Note that all traffic sent over those tunnels will be proxied over the Kubernetes controlplane.
A high amount of traffic and connections can increase pressure on the apiserver.
If you need regular high amount of connections and traffic, consider using a dedicated Kubernetes Service instead.

### Example

1. Create VM
```yaml
# ssh-test.vm.yaml
apiVersion: kubevirt.io/v1alpha3
kind: VirtualMachine
metadata:
  annotations:
    kubevirt.io/latest-observed-api-version: v1alpha3
    kubevirt.io/storage-observed-api-version: v1alpha3
    name.os.template.kubevirt.io/fedora32: Fedora 31 or higher
  name: ssh-test
  labels:
    app: ssh-test
    flavor.template.kubevirt.io/tiny: 'true'
    os.template.kubevirt.io/fedora32: 'true'
    vm.kubevirt.io/template: fedora-server-tiny-v0.11.3
    vm.kubevirt.io/template.namespace: openshift
    vm.kubevirt.io/template.revision: '1'
    vm.kubevirt.io/template.version: v0.12.4
    workload.template.kubevirt.io/server: 'true'
spec:
  running: false
  template:
    metadata:
      labels:
        flavor.template.kubevirt.io/tiny: 'true'
        kubevirt.io/domain: ssh-test
        kubevirt.io/size: tiny
        os.template.kubevirt.io/fedora32: 'true'
        vm.kubevirt.io/name: ssh-test
        workload.template.kubevirt.io/server: 'true'
    spec:
      domain:
        cpu:
          cores: 1
          sockets: 1
          threads: 1
        devices:
          disks:
            - disk:
                bus: virtio
              name: cloudinitdisk
            - bootOrder: 1
              disk:
                bus: virtio
              name: rootdisk
          interfaces:
            - masquerade: {}
              model: virtio
              name: nic-0
          networkInterfaceMultiqueue: true
          rng: {}
        machine:
          type: pc-q35-rhel8.2.0
        resources:
          requests:
            memory: 1Gi
      hostname: ssh-test
      networks:
        - name: nic-0
          pod: {}
      terminationGracePeriodSeconds: 180
      volumes:
        - cloudInitNoCloud:
            userData: |
              #cloud-config
              user: fedora
              password: ssh-demo
              chpasswd:
                expire: false
          name: cloudinitdisk
        - containerDisk:
            image: kubevirt/fedora-cloud-container-disk-demo
          name: rootdisk
```
```sh
kubectl apply -f ssh-test.vm.yaml
```

2. Start VM
```sh
kubectl virt start ssh-test
```

3. SSH into VM
```sh
kubectl virt ssh --username=fedora ssh-test
```
or
```sh
ssh -o 'ProxyCommand=kubectl virt port-forward --stdio=true ssh-test 22' fedora@ssh-test.default
```
