# Layer two connectivity for nested k8s hosts

The purpose of this proposal, is to allow connecting VMs to the node network,
in contrast to
[docs/libvirt-pod-networking.md](docs/libvirt-pod-networking.md), which allows
VMs to be connected to the Kubernetes Overlay network. As a consequence there
are two main points to consider:

 1) How and where to specify what the master host network device to connect to
    is.
 2) How to bring a device in position, so that libvirt can use it.

## How to specify the host network device

Right now, adding an extra network type `NodeNetwork` will be added. It does
not take any extra arguments. On the node, the plugin which creates a new
endpoint for the device will get the node device name passed in by
virt-handler. Virt-handler, which has to invoke the plugin from the host
network namespace view, will fetch the node IP from the `Node` object in the
cluster, and derive the device from it.

The section on the `Node` from which the IP needs to be taken:

```yaml
status:
  addresses:
  - address: 192.168.121.180
    type: InternalIP
```

Based on this IP, the master device can be derived and passed in to the plugin.
It should be sufficient to derive this information once, on virt-handler start.

When a VM contains a network configuration like this:

```yaml
kind: VM
spec:
  domain:
    devices:
      interfaces:
      - type: NodeNetwork
        mac:
          address: 00:11:22:33:44:55
        model:
          type: virtio
```

The plugin get's passed in the following config:

```json
{
    "name": "myvm-nodenetwork",
    "type": "macvlan",
    "master": "eth0",
    "ipam": {}
}
```

which will do the mapping. No IPAM module is specified, since the VM can use
plain DHCP to get a host network IP.

Finally, to forbid unautorized users to connect the host networks, we can think
about a similar concept like a `PodSecurityPolicy`, which can be tied to RBAC
access.

## macvlan/macvtap

```
  Host               Container                               VM
| ---------------- | ------------------------------------- | ------------ |
|           KubeVirt CNI          |                     Libvirt           |
| ------------------------------- | --------------------------------------| 

  eth0 ----------------macvlan1 -------> macvtap1 -------------> /dev/ethX

```

virt-handler calls a KubeVirt CNI plugin. The plugin creates a veth interface,
moves one end into the container, and connects the other end with the host
interface. The plugin is also responsible for generating/providing MAC
addresses, if none are provided. The plugin returns the name of the veth
endpoint in the container, and the MAC address. Libvirt then uses it's
`direct` attach capability, by creating a macvtap interface, and configures it
with the MAC address provided.

To check: 

 * Does libvirt need udev in the container or does it the work itself?

Advantage:

 * At the end macvtaps are used, which means that the VM get's the mac address
   assigned by libvirt

Disadvantage:

 * The VM can't directly communicate with the underlying node, if switches are
   not configured to mirror traffic.

Based on this proposed method, at the end, one would see


```xml
<interface type='direct'>
  <source dev='macvlan1'/>
</interface>
```

in the final domain XML.

## Other Approaches considered

The following sections are general thoughts about different more or less
possible solutions, which were also considered.

### macvtap

```
  Host               Container                               VM
| ---------------- | ------------------------------------- | ------------ |
|           KubeVirt CNI                         |      Libvirt           |
| ---------------------------------------------- | -----------------------| 

  eth0 --------------------------> macvtap ------------> /dev/ethX

```

virt-handler calls a KubeVirt CNI plugin. The plugin creates a veth interface,
moves one end into the container, and connects the other end with the host
interface.  The plugin is also responsible for generating/providing MAC
addresses, if none are provided. Then it attaches a macvtap device to the veth
endpoint inside the container and configures the appropriate MAC address for
it. The plugin returns the name of the macvtap endpoint in the container, and
the MAC address. Since libvirt does not allow direct passthrough of
pre-configured macvtap devices, the device would be directly attached via
passed in qemu arguments.

Advantage: 

 * The KubeVirt CNI can do the whole container side setup, including device
   node creation and cgroup setup. Libvirt would be a pure consumer. 

Disadvantage:

 * No libvirt support
 * The VM can't directly communicate with the underlying node, if switches are
   not configured to mirror traffic.

### tap/Bridge

Normally with the Kubernetes overlay networks, bridges are not required or
configured. We would have to configure the bridge based on an existing device.
That is very hard to achive, since we don't own the host network. If on the
host, a bridge is used, veth pairs can be used. Libvirt then attaches the
macvtap device to the veth endpoint inside the libvirt container.

```
  Host               Container                               VM
| ---------------- | ------------------------------------- | ------------ |
|           KubeVirt CNI          |                     Libvirt           |
| ------------------------------- | --------------------------------------| 

  bridge -> veth0 === veth1 --------> macvtap -------------> /dev/ethX

```

Advantage is, that the VM can talk to the node it runs on, like with every
other host in the host network.
