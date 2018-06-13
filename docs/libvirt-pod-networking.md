# Using CNI Interfaces and libvirt for networking

Author: Fabian Deutsch \<fabiand@redhat.com\>

## Introduction

Today, networking in KubeVirt is not integrated with Kubernetes. In order to
have network connectivity inside the VMI, a user must create a specific VMI Spec
that induces a particular libvirt domxml, in order to reuse a pre-exising host
network interface.

This is obviously suboptimal:

- It is not integrated with Kubernetes
- It does not provide the VMI with connectivity to pods
- It requires to specify host specific bits (the network device to use)

The purpose of this document is to describe one approach of how we can provide
a minimal integrated solution, with few workarounds.

**Note:** This approach is rough, and not optimized, but the rule of thumb was
to find a robust solution which works with Kubernetes today.


## Use-case

The primary use-case is to allow VMI to get connected to the pod network.


## API

The API is pretty clear: As usual an `interface` element will be defined.
To indicate that pod interfaces shall be used for networking, a new network
type (`pod-network`) is introduced.
All other fields, concerning the properties of the interface, can be used as
usual.

```yaml
kind: VirtualMachine
spec:
  domain:
    devices:
      interfaces:
      - type: pod-network
        mac:
          address: 00:11:22:33:44:55
        model:
          type: virtio
```


## Implementation

Prerequisites:

- libvirt is using pod networking (instead of its currently using host
  networking), as future Kubernetes multi-networking will work with pods,
  and as we do not want to mess with the hosts network namespace
- A CNI proxy to allow pods to perform CNI actions in the context of the host
  (this needs to be implemented, as part of this feature, or as an dependent
  one)

Conceptually the implementation works by assuming that every vNIC will be
mapped to a _new_ pod interface. To achieve this, for every vNIC a _new_ pod
interface is requested through CNI.
Another crucial constraint for this design is that in Kubernetes IP addresses
are getting assigned to pods, not interfaces. Thus this solution needs to
achieve the same, and provide IP addresses to VMIs, not just a interface.

During this document we call each of these _new_ pod interfaces _VMI interface_,
in order to differentiate them from the the originally-allocated pod interface
(`eth0`). The original pod interface (`eth0`) is never modified by Kubevirt,
and can be used to access libvirtd (through the libvirtd pod IP) or to provide
VMI-centric services through the VMI pod IP.

The required steps to enable networking as described are:

1. Request/Create a _new_ VMI interface on the libvirt pod, by using the CNI
   proxy to use the hosts CNI configuration and plugins.
2. Remember the IP of the VMI interface, then remove the IP from the VMI
   interface
3. Create a libvirt network, backed by a bridge, enable DHCP,
   and configure libvirt to provide the remembered IP address to the VMI.
   Achieved by adding a DHCP host entry, which maps the vNIC MAC address to the
   remembered IP
4. On VMI shutdown, delete all VMI interfaces. They can be infered by following
   the libvirt networks associated with the vNICs

**Note:** The _new_ VMI interface can not be directly attached to the VMI,
because we lose the ability to provide the IP to the guest. The IP can only be
provided by DHCP, which requires the use of a bridge.

This process must be repeated for every virtual NIC of every VMI.
Thus N virtual NICs correspond to N VMI interfaces, and in turn to N IP
addresses.

**Note:** Creating one VMI interface per virtual NIC might look like an
overhead, but this is a Kubernetes/CNI-compatible way to request IP endpoints
from the networking sub-system.


## Drawbacks & Limitations

* Every virtual NIC can only use a single IP address (the one provided via
  DHCP)
* A new VMI interface is required for every vNIC.
* Cleanup of the VMI interfaces in case of an uncontrolled pod shutdown is
  currently not handled


## Benefits & Opportunities

* Biggest plus is that this solution is using Kubernetes infrastructure for
  networking, is pretty much on the KubeVirt side (except for calling CNI on
  the host side), it is also reusing much of libvirt's network functionality.
* If the CNI plugins are modified to signal if they can also provide L2
  connectivity, then there is the chance, that a very similar mechanism can be
  used in future to provide L2 connectivity to VMIs (i.e. with the bridge or
  vxlan plugins).


# Future development

This approach is to provide initial, Kubernetes-friendly network connectivity
for virtual machines.
At this moment in time it is not clear if this method can be used to cover more
advanced use cases, like NIC passthrough, L2 connectivity, SR-IOV etc.
Thus for now the assumption is that this method is not suited to cover them, and
that additional methods are needed in future.
