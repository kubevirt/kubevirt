# Layer two connectivity for nested k8s hosts

The purpose of this proposal, is to allow connecting VMs to the node network,
in contrast to
[docs/libvirt-pod-networking.md](docs/libvirt-pod-networking.md), which allows
VMs to be connected to the Kubernetes Overlay network. As a consequence there
are three main points to consider:

 1) How and where to specify what the master host network device to connect to
    is.
 3) How can the nested nodes and the real node communicate with each other.
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
    "mode": "bridge",
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
  <source dev='macvlan1' mode='bridge'/>
</interface>
```

in the final domain XML.

### How can the node and the nested node communicate with each other?

In case of `macvlan` and `macvtap`, normally hosts and guests/containers can't
communicate with each other. However, we can add an additional `macvlan`
interface, bound to the same host interfaces like all other `macvlan` devices.
We can then route all host traffic through that `macvlan` device, to allow host
to guest and guest to host communication ([1], [2]).

In principle `virt-handler` would on startup do the following setup:

```bash
DEV=eth0
DEV_IP=$(ip addr show $DEV | grep -Po 'inet \K[\d.]+')
DEV_SUBNET=$(echo "$DEV_IP" | cut -d"." -f1-3)
# Assuming that $DEV is also responsible for the default gateway,
# and we don't use dhcp, we have to modify the default route later on
GATEWAY=$(ip -o route | grep default | awk '{print $3}')

# add the namespaces where we place the ipvlan for test purposes
ip netns add ns1

# create the macvlan link attaching it to the parent host $DEV
ip link add mv1 link $DEV type macvlan mode bridge

# move the new interface mv1 to the new namespace
ip link set mv1 netns ns1

# bring the interface up
ip netns exec ns1 ip link set dev mv1 up

# set ip addresses
ip netns exec ns1 ifconfig mv1 $DEV_SUBNET.11/24 up

# Create a host ipvlan through which the host and the guests can talk to each other
ip link add mv4 link $DEV type macvlan mode bridge
ip link set dev mv4 up


# here we multiple options:
# * use dhcp, it will do the default routes for us
# * do dhcp without setting the gateway, we then have to set it manually
# * unconfigure $DEV and set the IP on mv4 and manully change the default route
# using an extra IP via dhcp sounds like the best solution.
# We can leave all other routes which belong to $DEV untouched.
ifconfig mv4 $DEV_SUBNET.13/24 up
ip route del default
ip route add default via $GATEWAY dev mv4

# Remove the now wrong route for $DEV. We want all normal host traffic now go through mv4
ip route del $DEV_SUBNET.0/24 dev $DEV

# Add an extra route which allows accessing the $DEV_IP through mv4
ip route add $DEV_IP/32 via $DEV_SUBNET.13 dev mv4

# ping from container to host
ip netns exec ns1 ping $DEV_SUBNET.13
```

In this scenario, we can keep the IP of our main interface in place, and only
have to change two routes. All other routes which need the main interface can
be left untouched. In theory, moving the IP of the main interface to `mv4`
would also be possible, but that would require extensive route rewrites which
will likely cause conflicts with other components involved.

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

# References

[1] https://www.furorteutonicus.eu/2013/08/04/enabling-host-guest-networking-with-kvm-macvlan-and-macvtap/
[2] http://www.flat-planet.net/?p=479
