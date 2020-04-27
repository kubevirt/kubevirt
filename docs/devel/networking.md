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
is split into two distint phases:
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
required by virt-launcher to configure networking is the `CAP_NET_ADMIN`
capability.

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
creates an in-pod bridge, and sets the pod networking interface as a slave to
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
have the in-pod bridge as its master, and the tap device's MAC address, and
link MTU will be configured according to the values set in the domain xml.

Finally, and depending if the pod networking interface had configured IP
address(es), an in-pod DHCP server will be created to advertise the IP address
and routes which are cached on the VIF structure. This last step effectively
carries over the pod interface configuration to the VM interface, transparently
to the user. 

When the pod networking interface does not feature an IP address, the in-pod
DHCP server will not be started, leaving the VM with plain L2 connection via
the in-pod bridge.
