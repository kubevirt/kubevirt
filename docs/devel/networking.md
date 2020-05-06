# VMI Networking

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
