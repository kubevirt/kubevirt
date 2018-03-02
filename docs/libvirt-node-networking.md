# Layer Two Connectivity for nested k8s Nodes

The purpose of this proposal, is to allow connecting VMs to the node network,
in contrast to
[docs/libvirt-pod-networking.md](docs/libvirt-pod-networking.md), which allows
VMs to be connected to the Kubernetes Overlay network. This allows VMs started
by KubeVirt, to act as full k8s nodes themselves. Inside the VMs, the overlay
network provider used in the k8s installation will work as expected and the
nested node looks like an ordinary node from networking perspective.

There are three main points to consider, to achive this:

 1) How and where to specify what the master host network device to connect to
    is.
 2) How can the nested nodes and the real node communicate with each other.
 3) How to bring a device in position, so that libvirt can use it.
 
 Further the proposal focuses on two network topologies:
 
 1) The master host network device is **not** a **bridge**
 2) The master host network device is a **bridge**.
 
 In both cases, the goal is to keep the host network as much as it is. Thus the
 most optimal solution only adds network related object (device, routes, ...),
 but does not modify existing configurations.

 For both network topologies, the existence of a DHCP service for the nodes
 network is required.

## How to specify the host network device

Right now, adding an extra network type `NodeNetwork` will be added. It does
not take any extra arguments. On the node, the CNI plugin which creates a new
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

Based on that IP we can derive the ethernet device which we need. A simple bash
script like
```bash
DEV=$(ip route | grep "src 192.168.121.180" | sed -e "s/.*dev \([^ ]*\).*/\1/g")
echo $DEV
eth0
```
illustrates that.

Additionally annotations or labels can be considered, to override the detection
procedure, and directly use a specified device.

## Non-Bridge Scenario (macvlan/macvtap)

```
  Host               Container                               VM
| ---------------- | ------------------------------------- | ------------ |
|           KubeVirt CNI          |                     Libvirt           |
| ------------------------------- | --------------------------------------| 

  eth0 ----------------macvlan1 -------> macvtap1 -------------> /dev/ethX
```

virt-handler calls a KubeVirt CNI plugin. The plugin creates a macvlan
interface, moves it into the container, and attaches the macvlan interface to
the host interface. The plugin is also responsible for generating/providing MAC
addresses, if none are provided. The plugin returns the name of the veth
endpoint in the container, and the MAC address. Libvirt then uses it's `direct`
attach capability, by creating a macvtap interface, and configures it with the
MAC address provided.

To check: 

 * Does libvirt need udev in the container or does it the work itself?

Advantage:

 * At the end macvtaps are used, which means that the VM get's the mac address
   assigned by libvirt
 * In theory macvlans and macvtaps are faster than bridges.
 * We can do networking this way, on pretty much every k8s cluster out there,
   without any specific needs on the host network.

Disadvantage:

 * The VM can't directly communicate with the underlying node, if switches are
   not configured to mirror traffic. However, a non intrusive work-around will
   be scetched below.

### How can the node and the nested node communicate with each other?

In case of `macvlan` and `macvtap`, normally hosts and guests/containers can't
communicate with each other. However, we can add an additional `macvlan`
interface, bound to the same host interfaces like all other `macvlan` devices.
We can then route all guest traffic which needs to reach thoe node via it's
node IP through that `macvlan` device, to allow host to guest and guest to host
communication ([1], [2]).

To illustrate that, let's suppose that we have the follwoing scenario:

```
| host                     | guest                    |
|--------------------------|--------------------------|
| eth0 (192.168.200.1)     | macvtap1 (192.168.200.3) |
| macvlan0 (192.168.200.2) |                          |
```

`macvlan0` and `macvtap1` are both attache to `eth0`. Therefore, `macvlan0` and
`macvtap1` can both talk to each other. When we are on the host, we can send
and receive traffic via `eth0` and `macvlan0`. However, when we are in the
guest, we can't communicate with `eth0` via `macvtap1`.

We can now make use of the fact, that `macvtap1` can talk to `macvlan0` and add
a route on the host, which tells the kernel, to rout traffic to the IP
`192.168.200.1` via `192.168.200.2`. Once the packages are received by the
kernel from `macvlan0`, the kernel sees that they should go to `eth0` on this
host and forwards them.

The corresponding routing rule looks like this:

```
ip route add 192.168.200.1/32 via 192.168.200.2 dev macvlan0
```

For every VM we start we add such a rule, for every VM we stop we remove the
rule. No changes of existing routes are necessary.

#### Prepare the host network

`virt-handler` would on startup do the following setup:

First, assuming that `eth0` was detected as the main node interface, CNI will
be invoked with the follwoing parameters:
```json
{
    "name": "myvm-nodenetwork",
    "type": "macvlan",
    "master": "eth0",
    "mode": "bridge",
    "ipam": {
    		"type": "dhcp"
    	}
}
```
that CNI plugin, which can be taken from the CNI example plugins, roughly does

```bash
DEV=eth0
DEV_IP=$(ip addr show $DEV | grep -Po 'inet \K[\d.]+')
DEV_SUBNET=$(echo "$DEV_IP" | cut -d"." -f1-3)

# Create a host ipvlan through which the host and the guests can talk to each other
ip link add macvlan0 link $DEV type macvlan mode bridge
ip link set dev macvlan0 up

# here we multiple options:
# * use dhcp, it will do the default routes for us
# * do dhcp without setting the gateway, we then have to set it manually
# * unconfigure $DEV and set the IP on macvlan0 and manully change the default route
# using an extra IP via dhcp sounds like the best solution.
# We can leave all other routes which belong to $DEV untouched.
# No matter what we do, we don't want any routes modified.
ip addr add $DEV_SUBNET.13/24 dev macvlan0 noprefixroute

# normally we already have the netork namespace of libvirt given.
ip netns add ns1

# create the macvlan link attaching it to the parent host $DEV
ip link add macvlan1 link $DEV type macvlan mode bridge

# move the new interface macvlan1 to the new namespace
ip link set macvlan1 netns ns1

# bring the interface up
# ns1 will later be used by libvirt to attach macvtaps to
ip netns exec ns1 ip link set dev macvlan1 up
```

After that invocation, we have a host macvlan, with an IP address assigned. The
CNI dhcp plugin comes with a minimalistic dhcp client, which we can use to do
the DHCP renewals. An important side note here is, that the minimalistic daemon
returns the acquired IP alog side with routing rules suggested by the DHCP
server, but they are not applied, so the host is not modified.

#### Prepare the VM device

When a VM contains a network configuration like this:

```yaml
kind: VM
spec:
  domain:
    devices:
      interfaces:
      - type: HostNetwork
        mac:
          address: 00:11:22:33:44:55
        model:
          type: virtio
```
For every `HostNetwork`, `virt-handler` let's libvirt create a new macvtap
attached to the one prepared macvlan in the initial setup step in the libvirt
namespace.  Ee will use the CNI `dhcp` IPAM plugin with a provided or generated
MAC address. Based on this MAC address and the IPAM plugin invocation we end up
with the correct IP and MAC pair. The MAC address will then be used by libvirt
as the mactap address. The IP can be used to add a route, to allow routing
between the new guest and the host.

Taking the macvlan name and the MAC address, we can now transform the
HostNetwork entry into a libvirt XML fragment:

```xml
<interface type='direct'>
  <mac address="fe:54:00:f7:78:6d"/>
  <source dev='macvlan1' mode='bridge'/>
</interface>
```

When the guest now tries to acquire an IP address with this mac address, it
will get the same IP address. Based on that assumption we can create the route
to allow the guest to talk to the node.

```bash
# Add an extra route which allows accessing the $DEV_IP through the master macvlan.
# We need one of these routes per device.
ip route add $DEV_SUBNET.11/32 via $DEV_SUBNET.13 dev macvlan0

# ping from container to host
# done by the ipam dhcp client
ip netns exec ns1 ping $DEV_SUBNET.13
```

In this scenario, we can keep the IP of our main interface in place, and only
have to add one additional route per acquired IP for a VM. All existing routes
can stay untouched. In theory, moving the IP of the main interface to `macvlan0`
would also be possible, but that would require extensive route rewrites which
will likely cause conflicts with other components involved.

### Bridge Scenario (tap)

Normally with the Kubernetes overlay networks, bridges are not required or
configured. We would have to configure the bridge based on an existing device.
That is very hard to achive, since we don't own the host network.

If, howerverr on the host, a bridge is used, veth pairs can be taken, to allow
libvirt to attach VMs to the bridge. Libvirt then attaches the macvtap device
to the veth endpoint inside the libvirt container.

```
  Host               Container                               VM
| ---------------- | ------------------------------------- | ------------ |
|           KubeVirt CNI          |                     Libvirt           |
| ------------------------------- | --------------------------------------| 

  bridge -> veth0 === veth1 ------------> tap -------------> /dev/ethX

```

Advantage is, that the VM can talk to the node it runs on, like with every
other host in the host network. No special considerations are necessary.

So, when a VM contains a network configuration like this:

```yaml
kind: VM
spec:
  domain:
    devices:
      interfaces:
      - type: HostNetwork
        mac:
          address: 00:11:22:33:44:55
        model:
          type: virtio
```

We create a veth pair, attach one end to the bridge and move the other end into
the libvirt namespace. Libvirt can then attach the VM to the veth endpoint via
this domain XML snippet:

```xml
<interface type='bridge'>
  <source bridge='veth1'/>
  <mac address="00:11:22:33:44:55"/>
</interface>
```

# References

[1] https://www.furorteutonicus.eu/2013/08/04/enabling-host-guest-networking-with-kvm-macvlan-and-macvtap/
[2] http://www.flat-planet.net/?p=479
