This document explains on how to add the support for OvS-DPDK with kubevirt.

# Architecture

* [OpenvSwitch with
DPDK](http://docs.openvswitch.org/en/latest/intro/install/dpdk) enhances the
packet processing performance of the applications, bypassing the kernel space
with the help vhostuser sockets. It requires hugepages to be enabled which
will be shared between the host and guest for packet processing. This document
details the steps required to run OvS-DPDK on the host.

* [userspace CNI](https://github.com/intel/userspace-cni-network-plugin) plugin
which will be added a additional network to the multus meta plugin. It allows
us to create the vhostuser socket, which will be share between OpenvSwitch and
kubevirt (Qemu).

* In order to share vhostuser sockets between OpenvSwitch and qemu, a common
group `hugetlbfs` has been created by OpenvSwitch. Libvirt has to be
configured to start qemu-kvm with this group, so that the vhostuser socket can
be shared between them. OvS-DPDK supports two vhostuser modes -
`dpdkvhostuser` and `dpdkvhostuserclient`. The older mode `dpdkvhostuser`
where OpenvSwitch acts as a server, OvS creates the vhostuser socket file and
qemu acts as the client. This mode has been deprecated in OvS for reason that
in dynamic reconnection is not supported in qemu. Restarts of OvS requires the
VM to be restarted, which is a drawaback of this mode. This drawback has been
overcommed in the `dpdkvhostuserclient` mode, in which qemu acts as a server,
qemu creates the vhostuser socket file and OvS acts as a client. Restarts of
OvS does not impact VM as OvS can connect back to the vhostuser socket after
restart. Since the older mode `dpdkvhostuser` is deprecated, this
implementation will focus only on the `dpdkvhostuserclient` mode.

* IOMMU and hugepages has to be enabled in the nodes where OvS-DPDK is running.

# Configuration

In order to enable DPDK in OpenvSwitch, folllowing configurations has to be
enabled at different layers. For the simplicity of this document, it is assumed
that there is a running cluster with access to `oc` or `kubectl` command, which
has kubevirt enabled. In addition to this, following configuraiton are required:

## Cluster Configuration

OpenvSwitch will be running on the host, in order to avoid the default
OpenvSwitch container on the node where OvS-DPDK is enabled, configure a label
on the node to specify OvS is running externally.

```
oc label nodes worker-0 network.operator.openshift.io/external-openvswitch=""
```

It will ensure that OpenvSwitch container are excluded in the `worker-0` node by
the network operator.

> NOTE: As a next step, there should be an analysis to understand complexity in
> running OvS-DPDK in a container in the OpenShift or kubernets cluster.


## Node Configuration

Enable IOMMU and hugepages in the node. It is recommended to use 1GB hugepages
to obtain better performance. In order to enable IOMMU, add `intel_iommu=on
iommu=pt` to the kernel args. In order to enable 1GB hugepages, add
`hugepagesz=1G hugepages=64` to the kernel args. Here 64 counts of 1GB hugepages
are enabled in the node based on the node's available memory. Ensure the kernel
args are applied on the node:

```
[root@worker-0 ~]# cat //proc/cmdline
BOOT_IMAGE=/vmlinuz-3.10.0-1062.12.1.el7.x86_64 root=/dev/mapper/rhel_worker--0-root ro crashkernel=auto rd.lvm.lv=rhel_worker-0/root rd.lvm.lv=rhel_worker-0/swap rhgb quiet hugepagesz=1G hugepages=64 intel_iommu=on iommu=pt
```

To ensure the hugepages are enable succesfully, check the hugepage in the
meminfo:

```
[root@worker-0 ~]# cat /proc/meminfo | grep HugePages_
HugePages_Total:      64
HugePages_Free:       62

```

## OpenvSwitch Configuration

When DPDK is enabled in OvS, there will be Poll Mode Driver (PMD) thread, which
will be running in a indefinte loop to read and write the packets between the
interface driver and the VM. In order to achieve optimal performance, it is
required to isolate the CPUs used by PMD threads from other host processes,
other containers and any possible interrupts. Node tuning operator which uses
tuned to ensure this isolation, the require inputs should be provided to
isolate the CPUs. In this document, it is assumed that the required isolation
is done on the PMD cpus.

### OvS DB Configs

Configure the CPUs to be used for the PMD threads, it is recommende to use one
complete physical CPU (including siblings in case SMT) for optimal performance.
Below command configures the PMD mask as `c00000c`, which is derived from the
logical CPU numbers `2,26,3,27`. Two logical cores from each of the NUMA node is
added to PMD thread.

```
ovs-vsctl --no-wait set Open_vSwitch . other_config:pmd-cpu-mask=c00000c
```

Configure Socket memory to be used by the DPDK. For simplicity, we are assuming
1500 MTU for the socket memory requirement. A detailed guide will be linked to
this section on how to arrive at the memory values for various MTU values. 1GB
socket memory is alloacted on each of the NUMA node.

```
ovs-vsctl --no-wait set Open_vSwitch . other_config:dpdk-socket-mem=1024,1024
```

Enable DPDK with below configuration, which will restart `vswitchd` process by
enabling DPDK.

```
ovs-vsctl --no-wait set Open_vSwitch . other_config:dpdk-init=true
```

### OvS DPDK Port and Bridges

On a baremetal machine, bind the `vfio-pci` driver to the interface which will
be used for DPDK. Use `driverctl` tool to ensure this binding is persistent on
reboots.

Find the PCI address of the devices which will be used with DPDK
```
driverctl -v list-devices
```

Bind the PMD driver to the required interface
```
driverctl -v set-override 0000:06:00.0 vfio-pci
```

Create a OvS Bridge of `netdev` type
```
ovs-vsctl add-br br-dpdk0 -- set bridge br-dpdk0 datapath_type=netdev
```

Add the DPDK ports to the OvS bridge
```
ovs-vsctl add-port br-dpdk0 dpdk0 -- set Interface dpdk0 type=dpdk options:dpdk-devargs=0000:06:00.0
```

# Userspace CNI

Copy the userspace binary to the cni bin folder of the worker node. An
`EmptyDir` volume will used as the vhost socket directory in the virt-launcher
container. It will allow qemu to create the vhostuser socket in the empty
directory created at
`/var/lib/kubelet/pods/<podID>/volumes/kubernetes.io~empty-dir/shared-dir`.
The vhostuser socket will be created here.

The path with this empty directory and vhostuser socket name, goes beyond the
limt of 108 charcters for `sun_path` for the unix domain socket. If we add
this path as it is, DPDK will igore the full path and uses only 107 characters
for the socket path, resulting in file not found error. In order to avoid it,
mount the shared directory in to a local directory so that shrinked path can
be used, like `/var/lib/vhost_sockets/<podId>/`. Group ownership of this
directory will have `hugetlbfs`, so that the socket file created by qemu can
be used by OpenvSwitch.

# Create OvS-DPDK network-attachment-definition

Create a NetworkAttachmentDefinition resource for userspace cni with ovsdpdk
device type. Ensure the correct vhostuser mode is used (OpenvSwitch as client
and Qemu as server).

```
apiVersion: "k8s.cni.cncf.io/v1"                                                                                                                                      [3/382]
kind: NetworkAttachmentDefinition
metadata:
  name: userspace-ovs-net-1
spec:
  config: '{
        "cniVersion": "0.3.1",
        "type": "userspace",
        "name": "userspace-ovs-net-1",
        "kubeconfig": "/etc/kubernetes/cni/net.d/multus.d/multus.kubeconfig",
        "logFile": "/var/log/userspace-ovs-net-1-cni.log",
        "logLevel": "debug",
        "host": {
                "engine": "ovs-dpdk",
                "iftype": "vhostuser",
                "netType": "bridge",
                "vhost": {
                        "mode": "client"
                },
                "bridge": {
                        "bridgeName": "br-dpdk0"
                }
        },
        "container": {
                "engine": "ovs-dpdk",
                "iftype": "vhostuser",
                "netType": "interface",
                "vhost": {
                        "mode": "server"
                }
        },
        "ipam": {
                "type": "host-local",
                "subnet": "10.56.217.0/24",
                "rangeStart": "10.56.217.131",
                "rangeEnd": "10.56.217.190",
                "routes": [
                        {
                                "dst": "0.0.0.0/0"
                        }
                ],
                "gateway": "10.56.217.1"
        }
    }'

```

# Create VM with OvS-DPDK

Create a VM with `vhostuser` interface type. Ensure hugepages is added to the
VMI spec.

```
apiVersion: kubevirt.io/v1alpha3
kind: VirtualMachineInstance
metadata:
  name: vm-trex-1
spec:
  terminationGracePeriodSeconds: 30
  domain:
    cpu:
      sockets: 1
      cores: 8
      threads: 2
      dedicatedCpuPlacement: true
    machine:
      type: q35
    resources:
      requests:
        memory: 6Gi
    memory:
      hugepages:
        pageSize: "1Gi"
    devices:
      disks:
      - name: local-storage
        disk:
          bus: virtio
      - disk:
          bus: virtio
        name: cloudinitdisk
      interfaces:
      - name: default
        masquerade: {}
      - name: vhost-user-net-1
        vhostuser: {}
  networks:
  - name: default
    pod: {}
  - name: vhost-user-net-1
    multus:
      networkName: userspace-ovs-net-1
  volumes:
  - name: cloudinitdisk
    cloudInitNoCloud:
      userData: |-
        #cloud-config
        password: fedora
        chpasswd: { expire: False }
  - name: local-storage
    persistentVolumeClaim:
      claimName: local-pvc-2
```
