# VMI Networking
Before we explain how KubeVirt performs VM networking configuration, it is
paramount we separate the pod networking configuration and the  VM networking
configuration concepts. The underlying configuration of the pod network
interface is out of the scope of this document; Kubernetes is responsible to
set up pod networking according to its configuration, via CNI. It is safe to
assume it will simply plug a configured network interface into the pod, and
through it, the pod connects to the outside world.

The relevant part for KubeVirt is VM networking configuration, which will be
casually referred to as `binding` throughout this developer's guide.

## VMI networking configuration
In this section we'll explain how VM networking is configured. In order to
follow the principle of least privilege (where each component is limited
to only its required privileges), the configuration of a KubeVirt VM interfaces
is split into two distinct phases:
- [privileged networking configuration](#privileged-vmi-networking-configuration): occurs in the virt-handler process
- [unprivileged networking configuration](#unprivileged-vmi-networking-configuration): occurs in the virt-launcher process

### Privileged VMI networking configuration
Virt-handler is a trusted component of KubeVirt; it runs on a privileged
daemonset. It is responsible for creating & configuring any required network
infrastructure (or configuration) required by the
[binding mechanisms](#binding-mechanisms).

It is important to refer that while this step is performed by virt-handler, it
is performed in the *target* virt-launcher's net namespace.

The first action done in the privileged VMI networking configuration is to
identify which is the correct [BindMechanisms](#binding-mechanisms) to use.
Once the `BindMechanism` is defined, the correct implementation will perform
the following operations, in this order:

- `discoverPodNetworkInterface`: Each `BindMechanism` requires different
   information about the pod interface - slirp, for instance, doesn't require
   any info. The others, gather the following information:
   - IP address
   - Routes (**only** bridge)
   - Gateway
   - MAC address (**only** bridge)
   - Link MTU

- `preparePodNetworkInterfaces`: this function will make use of the
  aforementioned information, performing actions specific to each `BindMechanism`.
  See each `BindMechanism` section for more details.

- `setCachedInterface`: caches the interface in memory

- `setCachedVIF`: this will persist the VIF object in the file system, making
  the configured pod interface available to the virt-launcher pod.
  The VIF is cached in
  `/proc/<virt-launcher-pid>/root/var/run/kubevirt-private/vif-cache-<iface_name>.json`.

### Unprivileged VMI networking configuration
The virt-launcher is an untrusted component of KubeVirt (since it wraps the
libvirt process that will run third party workloads). As a result, it must be
run with as little privileges as required. As of now, the only capability
required by virt-launcher to configure networking is the `CAP_NET_ADMIN`.

In this second phase, virt-launcher also has to select the correct
`BindMechanism`, and afterwards will uses it to retrieve the configuration
data previously gathered in phase #1 (by loading the cached VIF object).

With the VIF information, it will proceed to decorate the domain xml
configuration of the VM it will encapsulate. This *decoration* is specific
to each [binding mechanism](#binding-mechanisms), as each of them will involve
different libvirt configurations.

The aforementioned flow translates into the following calls of the specific
`BindMechanism` implementations:

- `loadCachedInterface`
- `loadCachedVIF`
- `decorateConfig`

## Binding Mechanisms
A binding mechanism can be seen as the translation service between KubeVirt's
API and Libvirt's domain xml. It also features methods to set up the required
networking infrastructure.

Each interface type has a different binding mechanism, since it will lead to a
different libvirt domain xml specification. It may also require different
networking infrastructure to be created / configured - e.g. a bridge for the 
`bridge` or `masquerade` `BindMechanism`s. 

Code-wise, a `BindMechanism` in an interface which collects the following methods:

```golang
type BindMechanism interface {
	discoverPodNetworkInterface() error
	preparePodNetworkInterfaces() error

	loadCachedInterface(pid, name string) (bool, error)
	setCachedInterface(pid, name string) error

	loadCachedVIF(pid, name string) (bool, error)
	setCachedVIF(pid, name string) error

	// The following entry points require domain initialized for the
	// binding and can be used in phase2 only.
	decorateConfig() error
	startDHCP(vmi *v1.VirtualMachineInstance) error
}
```

As of now, the existent binding mechanisms are:
- [bridge](#bridge-binding-mechanism)
- [masquerade](#masquerade-binding-mechanism)
- [slirp](#slirp-binding-mechanism)

### Bridge binding mechanism
Using the bridge `BindMechanism` requires a VMI configuration featuring a
network whose interface type is `bridge` - the yaml file below can be used
as reference, but please refer to the
[user guide](https://kubevirt.io/user-guide/#/creation/interfaces-and-networks?id=bridge)
for more information.
```yaml
kind: VM
spec:
  domain:
    devices:
      interfaces:
        - name: default
          bridge: {}
  networks:
  - name: default
    pod: {} # Stock pod network
```

Let's refer to the image below to get a better understanding of how this
`BindMechanism` works.

Bridge binding mechanism diagram

![alt text](https://github.com/kubevirt/kubevirt.github.io/blob/source/assets/images/diagram.png)

As can be seen in the diagram above, there are three actors at play: CNI,
libvirt, and DHCP. For completeness sake, let's add one more actor that is
implicit in the picture: KubeVirt.

As indicated in [the introduction](#vmi-networking), the pod networking
configuration step - performed by CNI - is out of scope of this guide. The
focus will be instead on how KubeVirt performs bridge binding.

The bridge `BindMechanism` starts off by reading the pod networking interface
configured by CNI. It caches the assigned MAC address and the link MTU in the
VIF structure. When CNI has configured IP address(es) on the pod networking
interface, the bridge `BindMechanism` also caches the **first** IP address in
the VIF, along with any routes CNI has configured.

In the `preparePodNetworkInterfaces` method, KubeVirt randomizes the in-pod
veth mac-address; the original one, will be plugged into the VM. It also
creates an in-pod bridge, and sets the pod networking interface as a port to
this bridge. This happens in phase#1 - i.e. performed by virt-handler on
behalf of virt-launcher.

The `preparePodNetworkInterfaces` method performs one other operation: **if**
the pod networking interface featured any IP address, it will delete the first
one; remember this address is cached in the VIF structure It will be needed
for phase#2.

In phase#2 (executed by virt-launcher, unprivileged) the domain xml which will
create the VMI is generated. It first creates a rough domxml of each interface
of the VM, whose sole purpose is to instruct how to connect to the in-pod
bridge. The generated interface xml element looks like:

```xml
 <interface type='bridge'>
    <source bridge='k6t-eth0'/>
    <model type='virtio'/>
 </interface>
```

Afterwards, but still in phase#2, KubeVirt will decorate the aforementioned
interface xml, specifying both the link MTU and MAC address through the
`decorateConfig` `BindMechanism` method. In it, the MAC and MTU of the pod
interface are copied over to the interface dom xml definition, which, at this
stage, will look like:

```xml
<interface type='bridge'>
  <mac address='8e:61:55:c2:4a:bd'/>
  <source bridge='k6t-eth0'/>
  <target dev='vnet0'/>
  <model type='virtio'/>
  <mtu size='1440'/>
  <alias name='ua-bridge'/>
  <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
</interface>
```

Once the VM is booted, libvirt will consume the interface xml definition and
create a tap device - named after the `target` parameter. That tap device will
be attached to the in-pod bridge, and the tap device's MAC address,
and link MTU will be configured according to the values set in the domain xml.

Finally, and depending if the pod networking interface had configured IP
address(es), an in-pod DHCP server will be created to advertise the IP address
and routes which are cached on the VIF structure. This last step effectively
carries over the pod interface configuration to the VM interface, transparently
to the user. 

When the pod networking interface does not feature an IP address, the in-pod
DHCP server will not be started, leaving the VM with plain L2 connection via
the in-pod bridge.

### Masquerade binding mechanism
Similar to the [bridge bind mechanism](#bridge-binding-mechanism), triggering
the masquerade `BindMechanism` requires a VMI configuration featuring a
network with the `masquerade` interface type. KubeVirt provides an example of
a
[fedora based VMI](https://github.com/kubevirt/kubevirt/blob/main/examples/vmi-masquerade.yaml)
in the project's examples folder.

The masquerade bind mechanism has plenty in common with bridge binding; both
have virt-handler create an in-pod bridge, generate a similar looking interface
domain xml element - e.g. interface type *bridge*, and same *source* and
*target* values. Both phases of the networking configuration communicate by caching
data in the VIF structure.

However, the similarities end there; while the networking infrastructure is
the same, VM networking works completely different.

In masquerade binding, the goal is to NAT the traffic from the pod interface
into the VM interface via IPtables / NFtables. Before venturing into details,
the relevant knobs should be described.

There are 2 knobs that impact the configuration of the masquerade binding:
  - `ports`: an attribute of the interface, described in the
    `spec::domain::interfaces` subtree. Here the user indicates the allowlist
     of ports and protocols. It is important to mention that when the list is
     omitted, **all** ports are implicitly included.
  - `vmNetworkCIDR`: the CIDR from which the in-pod bridge **and** the VM will
    get their IP address. This attribute is defined in the `spec::networks`
    subtree. It defaults to `10.0.2.0/24`.

Please refer to the short example below to visualize the aforementioned knobs:
```yaml
...
spec:
  domain:
    devices:
      ...
      interfaces:
      - masquerade: {}
        name: masqueradenet
        ports:
        - name: http
          port: 80
          protocol: TCP
  networks:
  - name: masqueradenet
    pod:
      vmNetworkCIDR: 10.11.12.0/24
```

As with bridge binding, the `discoverPodNetworkInterface` caches the MTU of
the pod networking interface. It also assigns an IP address from the configured
CIDR (or the default), and reserves an IP for the gateway - which will be the
in-pod bridge. Re-using the example above, we would have `10.11.12.1` as
gateway IP, and `10.11.12.2` as VM IP.

NAT is configured in the `preparePodNetworkInterfaces` method. The bridge is
configured with the IP address previously reserved for the VM's gateway.

The bridge acts as the vm's default gateway and not as a L2 bridge,
therefore, the pod networking interface is not set as its port.
Since a linux bridge gets the MAC address of its first port and we don't want it to
take the MAC address of the first tap device attached to it,
`preparePodNetworkInterfaces` creates a dummy nic and sets it as the first
port of the bridge.

Afterwards, the nftables / iptables rules are provisioned in the NAT table. It
follows a standard one to one NAT implementation using netfilter.

It first involves the `PREROUTING` chain, which is responsible for packets that
have just arrived at the network interface. This rule simply filters all
incoming traffic from the pod networking interface, making it go through the
`KUBEVIRT_PREINBOUND` chain.

```
Chain PREROUTING (policy ACCEPT)
target               prot opt in   source               destination
KUBEVIRT_PREINBOUND  all  --  eth0 anywhere             anywhere
```

On the `KUBEVIRT_PREINBOUND` chain, packets will be DNAT'ed (have their
destination address changed) to the IP address of the VM - e.g. `10.11.12.2`.
This can be subject to an allowlist of ports (if one was provided by the user,
in the masquerade interface specification) or simply have all ports accepted -
when the port configuration is omitted.

```
Chain KUBEVIRT_PREINBOUND (1 references)
target     prot opt source               destination
DNAT       tcp  --  anywhere             anywhere             tcp dpt:http to:10.0.2.2
```

Before the packet leaves the interface, it will pass through the
`POSTROUTING` chain, which in turn, makes the packet go through the
`KUBEVIRT_POSTINBOUND` chain.

```
Chain POSTROUTING (policy ACCEPT)
target     prot opt source               destination
KUBEVIRT_POSTINBOUND  all  --  anywhere   anywhere
```

In the `KUBEVIRT_POSTINBOUND` chain, in case the source address is localhost, SNAT is
performed: the source IP address of the outbound packet is modified to the IP address of
the gateway -
`10.11.12.1`.

```
Chain KUBEVIRT_POSTINBOUND (1 references)
target     prot opt source               destination
SNAT       tcp  --  anywhere             anywhere             tcp dpt:http to:10.0.2.1
```

Once the packet leaves the interface, it will be subject to the routing tables
present on the virt-launcher pod. Since we've DNAT'ed the packet, it will be
routed to the in-pod bridge, which will forward the traffic to the VM.

All outbound traffic from the VM can reach the outside world via the in-pod
bridge. The packet will be routed via the default route -
`default via 169.254.1.1 dev eth0` - and before leaving the interface, its
source address will be masqueraded to the IP address of the pod, via the
masquerade target.

```
Chain POSTROUTING (policy ACCEPT)
target     prot opt source               destination
MASQUERADE  all  --  10.0.2.2             anywhere
```

### Masquerade binding using IPv6 addresses
The masquerade binding mechanism is currently the only binding mechanism which
accepts IPv6 addresses.

It operates in exactly the same way as in IPv4, and follows the same goals:
configure one to one NAT. As in it's IPv4 counter-part, the pods are reached
via their IPv6 pod addresses.

NAT is configured in the exact same way, but using ip6tables, or the `ipv6-nat`
nftable table. Please refer to the tables below to visualize how NAT for IPv6
addresses is accomplished in KubeVirt.

```
Chain PREROUTING (policy ACCEPT)
target               prot opt source               destination
KUBEVIRT_PREINBOUND  all      anywhere             anywhere

Chain INPUT (policy ACCEPT)
target     prot opt source               destination

Chain OUTPUT (policy ACCEPT)
target     prot opt source               destination
DNAT       tcp      anywhere             localhost            tcp dpt:http to:fd10:0:2::2

Chain POSTROUTING (policy ACCEPT)
target     prot opt source               destination
MASQUERADE  all      fd10:0:2::2          anywhere
KUBEVIRT_POSTINBOUND  all      anywhere   anywhere

Chain KUBEVIRT_POSTINBOUND (1 references)
target     prot opt source               destination
SNAT       tcp      anywhere             anywhere             tcp dpt:http to:fd10:0:2::1

Chain KUBEVIRT_PREINBOUND (1 references)
target     prot opt source               destination
DNAT       tcp      anywhere             anywhere             tcp dpt:http to:fd10:0:2::2
```

It is important to refer that masquerade binding configures NAT for both IPv4
and IPv6 address families - both address family traffic is forwarded into the
VM instance. Despite that, the only IP address reported is the IPv6 address,
which implicitly highly encourages IPv6 communication towards the VM instance.

On a final note, there is a difference that impacts the user experience when
using masquerade binding for IPv6 addresses; the VMI IP must be [manually
configured by the user](https://kubevirt.io/user-guide/virtual_machines/interfaces_and_networks/#masquerade-ipv4-and-ipv6-dual-stack-support).
