# Network Binding Plugin
[v1.4.0, Beta feature]

A modular plugin framework which integrates with Kubevirt to implement a
network binding.

This document is aimed at plugin developers which would like to create
a custom network binding and integrate it with Kubevirt.

Before proceeding with the reading of this developer guide, please read
the [Network Binding Plugins](https://kubevirt.io/user-guide/virtual_machines/network_binding_plugins/)
user guide.

## Overview

### Network Connectivity
In order for a VM to have access to external network(s), several layers
need to be defined and configured, depending on the connectivity characteristics
needs.

These layers include:

- Host connectivity: Network provider.
- Host to Pod connectivity: CNI.
- Pod to domain connectivity: Network Binding.

This guide focuses on the Network Binding portion.

### Network Binding
The network binding defines how the domain (VM) network interface is wired
in the VM pod through the domain to the guest.

The network binding includes:

- Domain vNIC configuration.
- Pod network configuration (optional).
- Services to deliver network details to the guest (optional).
  E.g. DHCP server to pass the IP configuration to the guest.

### Plugins
The network bindings have been part of Kubevirt core API and codebase.
With the increase of the number of network bindings added and
frequent requests to tweak and change the existing network bindings,
a decision has been made to create a network binding plugin infrastructure.

The plugin infrastructure provides means to compose a network binding plugin
and integrate it into Kubevirt in a modular manner.

Kubevirt is providing several network binding plugins as references.
The following plugins are available:

- [passt](https://kubevirt.io/user-guide/virtual_machines/net_binding_plugins/passt/) [v1.1.0]
- [macvtap](https://kubevirt.io/user-guide/virtual_machines/net_binding_plugins/macvtap/) [v1.1.1]
- [slirp](https://kubevirt.io/user-guide/virtual_machines/net_binding_plugins/slirp/) [v1.1.0]

## The Zero Code Plugin

The simplest network binding plugin is the one that requires no additional code.
It takes advantage of the
[Domain Attachment Type](https://kubevirt.io/user-guide/virtual_machines/network_binding_plugins/#domainattachmenttype)
option which provides a pre-defined core Kubevirt method to attach an interface
to the domain.

The currently supported domain attachment type is `tap` (v1.1.1) which
builds a domain interface configuration that points to the tap/macvtap
existing interface.

Such a binding plugin assumes that the CNI used for the network connectivity
exposes in the pod a `tap` or `macvtap` (type) interface with a name corresponding
to the hashed network name:

- `pod<hash network name>` (or plain `eth0` for the primary network)
- `tap<hash network name>` (or plain `tap0` for the primary network)

For secondary networks, Kubevirt virt-controller will define these names on the
pod Multus annotation (`"k8s.v1.cni.cncf.io/networks"`),
therefore there is no special action needed from the plugin author except
choosing a CNI that confirms with the CNI standard spec.

For clarification, the CNI used for the connectivity is the one defined
in the `NetworkAttachmentDefinition` object referenced from the VM network spec.

### Configuration example

Given the following registered binding plugin:

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    network:
      binding:
        my-zero-code-binding:
          domainAttachmentType: tap
```

The VM interface/network spec will include a ref to the binding plugin
and the `NetworkAttachmentDefinition` named `mynad`:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: myvm
spec:
  template:
    spec:
      domain:
        devices:
          interfaces:
            - name: mynetwork
              binding:
                name: my-zero-code-binding
      networks:
      - name: mynetwork
        multus:
          networkName: mmynad
```

The `mynad` NetworkAttachmentDefinition defines `cni-name` as the CNI plugin
binary:

> **Note**: The CNI plugin mentioned here is not related to the
> network binding plugin CNI (which is optional and described in detail later on).

```yaml
kind: NetworkAttachmentDefinition
apiVersion: k8s.cni.cncf.io/v1
metadata:
  name: mynad
spec:
  config: '{
      "cniVersion": "0.3.1",
      "name": "some-name",
      "type": "cni-name",
    }'
```

Kubevirt will eventually create a domain interface configuration that
points to the tap (e.g. `tap12345678`) or pod interface (e.g. `pod12345678`).
If the interface name that starts with `tap` does not exist, it will use the one
with `pod`.

In addition, the mac address of the pod interface is copied to the domain
configuration.

This is a snippet of the configuration:

```xml
<interface type='ethernet'>
   <alias name='ua-mynetwork'/>
   <target dev='pod12345678' managed='no'/>
   <model type='virtio-non-transitional'/>
   <mac address='12:34:56:78:9a:bc'/>
   <mtu size='1500'/>
   <rom enabled='no'/>
</interface>
```

A classic example of such a plugin is the
[macvtap](https://kubevirt.io/user-guide/virtual_machines/net_binding_plugins/macvtap/)
plugin.

## The Sidecar Plugin

When a standard domain attachment requires customization,
or when additional services are needed (e.g. DHCP), a sidecar container
may be executed to integrate with Kubevirt.

The sidecar container runs in parallel to the virt-launcher container
which runs the hypervisor (libvirt and qemu).

### Sidecar Protocol

Kubevirt supports an integration hook between the virt-launcher
and sidecar containers that run in the same pod.

The integration is based on a client-server gRPC protocol.
The server side is setup at the sidecar container and the client
operates at the virt-launcher container.

From the sidecar perspective, the operation steps are as follows:

- The sidecar container is defined as part of the virt-launcher pod,
  by the virt-controller. It includes a mounting point to
  `HookSocketsSharedDirectory`.

- The sidecar starts by listening on an existing unix socket.
```go
import "net"
import "os"
import "path/filepath"

import "kubevirt.io/kubevirt/pkg/hooks"

socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, hookSocket)
socket, err := net.Listen("unix", socketPath)
defer os.Remove(socketPath)
```

- Then registers handlers for various command requests:
  - Info: Returns information about the current server version and its supporting
    hook points. Clients use this information to correctly interact with the
    server (e.g. use the relevant requests supported).
  - Callback server: Registers the services the server supports.
    Depending on the version, the server side supports several operations exposed
    as methods. E.g. the `OnDefineDomain` method which expects a domain configuration
    and a VMI spec to be provided, returning an updated domain configuration as a result.

```go
import "google.golang.org/grpc"

import info "kubevirt.io/kubevirt/pkg/hooks/info"
import v1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"

server := grpc.NewServer([]grpc.ServerOption{}...)
info.RegisterInfoServer(server, srv.InfoServer{Version: "v1alpha2"})
v1alpha2.RegisterCallbacksServer(server, srv.V1alpha2Server{})
```

- Finally starting the gRPC server.
```go
server.Serve(socket)
```

From the client perspective (at the virt-launcher container), it calls
the `OnDefineDomain` command just after generating the domain configuration
and before sending the domain configuration to the hypervisor (i.e. libvirt).

Therefore, the sidecar may implement any logic that mutates the domain
configuration.
In the context of the network binding, the sidecar will usually add
a domain interface configuration.

### Sidecar logic

A network binding sidecar will usually mutate the domain configuration
based on the VMI specification.

The logic needs to be added to the `OnDefineDomain`.
It will usually have a structure similar to the following:

> **Note**: `changeDomain` is the function that accepts a domain and VMI spec,
> returning a new mutated domain spec.

```go
import "kubevirt.io/api/core/v1"
import hooksv1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
import domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

type V1alpha2Server struct{}

func (s V1alpha2Server) OnDefineDomain(_ context.Context, params *hooksv1alpha2.OnDefineDomainParams) (*hooksv1alpha2.OnDefineDomainResult, error) {
    vmi := &v1.VirtualMachineInstance{}
    if err := json.Unmarshal(params.GetVmi(), vmi); err != nil {
        return nil, fmt.Errorf("failed to unmarshal VMI: %v", err)
    }

    domainSpec := &domainschema.DomainSpec{
        // Unmarshalling domain spec makes the XML namespace attribute empty.
        // Some domain parameters requires namespace to be defined.
        // e.g: https://libvirt.org/drvqemu.html#pass-through-of-arbitrary-qemu-commands
        XmlNS: libvirtDomainQemuSchema,
    }
    if err := xml.Unmarshal(domainXML, domainSpec); err != nil {
        return nil, fmt.Errorf("failed to unmarshal given domain spec: %v", err)
    }

    updatedDomainSpec, err := changeDomain(domainSpec, vmi)
    if err != nil {
        return nil, err
    }

    newDomainXML, err := xml.Marshal(updatedDomainSpec)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal updated domain spec: %v", err)
    }

    return &hooksv1alpha2.OnDefineDomainResult{
        DomainXML: newDomainXML,
    }, nil
}
```

It is recommended to use the latest available hook version.
The unused methods must be defined to satisfy the interface, preferably with
a default non-error response.

#### Pod interface naming

The name of the network interface in the pod, to which the relevant network
is to bind to, is expected to be known.

- For the primary network, the default `eth0` naming is expected.
- For the secondary networks, Kubevirt explicitly specifies the name through
  the pods Multus annotation (`"k8s.v1.cni.cncf.io/networks"`).
  The CNI plugin is expected to use the interface name inputted to define the in-pod
  interface name.
  The name used is generated by Kubevirt with the following format:
  `pod<network name hash ID>`.
  - Network name hash ID: The hash ID is generated based on the network name specified
    in the VM/VMI interface & network spec.
    To programmatically generate it, one can use the following code snippet:

```go
import "kubevirt.io/kubevirt/pkg/network/namescheme"

ifaceName := namescheme.GenerateHashedInterfaceName(network_name)
```

#### Domain configuration

Changing the domain configuration usually involves the addition of a device
or a change in an existing device.

Either way, it is important to understand which are the networking
devices and what are the expectations from Kubevirt:

- Domain API: "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
  - `DomainSpec`: The domain configuration spec structure, which is in sync
    with [libvirt domain configuration](https://libvirt.org/formatdomain.html).
    - `Devices`:
      - `Interface`: The network interface configuration spec.
      - `HostDevice`: The general host device configuration spec.
        The host-device is used to pass-through devices from the host to the domain.
        SR-IOV virtual functions are a classic example for networking.

- Device Alias:
  Kubevirt marks on the domain device spec an alias that references
  the logical network name as appearing in the VM/VMI spec.
  - For SR-IOV devices, the alias also includes a prefix: `sriov-`.

  ```go
  import domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
  deviceAlias := domainschema.NewUserDefinedAlias("sriov-" + "mynet1")
  ```

- Parameters: Several configuration parameters of an interface from the VMI spec
  are passed to the domain device configuration directly (e.g. the mac address).
  The domain interface and host-device expose additional parameters
  which are not exposed through the VMI interface spec.
  Plugin authors may populate other domain parameters if needed, taking
  the values as hard-coded or from the VMI object (including annotation).

### Sidecar Artifacts

The expected artifacts include:

- Documentation on how to register it to Kubevirt and what it does.
- A container image that includes the logic of a network binding plugin sidecar.
- Optionally, a link to a registry that contains the container image.

> **Note**: In order to consume a sidecar, the image needs to be available in
> an accessible registry.
> Cluster admins may need to make the sidecar image available on their registry.

## The Binding CNI Plugin

We have covered so far network binding plugins that utilize core domain attachment
types and/or custom sidecars. While many network bindings can find these
integration points sufficient, there is a missing piece.

Network bindings that require pod network changes, should use a CNI plugin.

### Using a custom CNI plugin

A CNI plugin is a program that is called by the runtime when the pod is created.
The program is expected to interpret a standard [CNI](https://www.cni.dev/)
format input and perform changes in the pod network namespace.

> **Note**: The CNI plugin discussed in this section is used for implementing
> a network binding plugin, connecting the pod network to the guest.
> It is not to be confused with the CNI plugin that connects the pod to
> a network on the node.

#### Custom CNI plugin requirements

In order to call multiple times CNI plugins, the cluster requires
Multus to be deployed.

In addition, the CNI plugin itself needs to be deployed on the cluster nodes.

> **Note**: It is out of this document scope to elaborate on how a binary can
> be deployed on cluster nodes and kept up-to-date.

A network binding CNI plugin belongs to a binding, and it is
accompanied by a Network Attachment Definition (NAD) that references the
binary.

#### Creating a network binding CNI plugin

At the time of this writing, there is no known document or tutorial on how
to compose a CNI plugin. However, there are many CNI plugins around to use
as references.

We will take the same approach here and reference the `passt` binding CNI.
It is very simple to follow, performing minimal changes on the pod netns.

Ref: https://github.com/kubevirt/kubevirt/tree/release-1.1/cmd/cniplugins/passt-binding

> **Note**: A CNI plugin outputs its results to `stdout`, therefore avoid
> outputting logging to `stdout` (e.g. use of `fmt.Printf()`) as it will
> corrupt the results.

### Network Binding CNI Operation Flow

Understanding the details of how the binding CNI plugin integrates into a VM
creation may assist in its proper usage.

The following steps occur when an interface is defined with a network binding
plugin that uses a custom binding CNI:

- The `virt-controller` inspects the VMI spec and iterates over the network
  interfaces.
- For each interface, if a network binding plugin is used with
  a network-attachment-definition defined, the `virt-controller` will add to
  the `virt-launcher` pod network (multus) annotation a plugin entry that
  references the binding network-attachment-definition.
- The binding entry includes:
  - The network-attachment-definition name and namespace.
  - The `cni-args`, which allows passing arbitrary data to the CNI plugin.
    At the time of this writing, it is used to pass the logical network name.
    Identified by the field key: `logicNetworkName`.
- The `virt-controller` creates the `virt-launcher` pod and eventually the runtime
  calls the CNI interface with the combined data from the
  network-attachment-definition and pod network annotation.
- Multus acts as a CNI delegator, calling all CNI plugins, including Kubevirt
  network binding CNI plugins.
  - First, it calls the "regular" network CNI plugin which connects the pod to the
    relevant network on the pod.
  - Secondly, it calls the network binding CNI plugin, if it exists, to configure
    the pod networking per the binding needs.
  > **Note**: The network binding CNI entry in the pod network annotation is
  > injected **after** the "regular" connectivity entry.

### Network Binding CNI Logic

The CNI plugin used for the network binding is focused on performing
the needed changes in the pod network namespace.
It is important to avoid using it to perform changes on the host side.

In order to identify on which network interface context the plugin is called,
the `logicNetworkName` can be used.
From it, the possible pod interface name can be calculated:

```go
import "kubevirt.io/kubevirt/pkg/network/namescheme"

podIfaceName := namescheme.GenerateHashedInterfaceName(networkName)
```

It is possible to perform both interface specific and interface agnostic
changes in the pod netns. It is important to consider that there may be other
CNI plugins called in the chain which may collide with such changes.
Therefore, it is important to take into consideration these factors and try
to limit the changes to specific kernel interfaces as much as possible.

### Network Binding CNI and sidecars

The need to share data from the CNI stage to the `virt-launcher` stage may arise.
There may be several options to implement this, one relatively simple
method could be to preserve the data on a `dummy` interface.
Aa a network binding plugin author, both the CNI plugin and the sidecar codebase
is available, therefore both can be in sync to share such information.

## Migration Support

The VM migration operation is considered a core feature in Kubevirt and
therefore the ability to support it needs to be addressed when designing
a network binding plugin.

The following points should be addressed in relation to migration support:
- Kubevirt will allow migration only if the primary network supports it.
- For a network binding plugin to support migration, the `migration` `method`
  must be specified in the Kubevirt CR.
  See the user-guide network binding plugin
  [migration](https://kubevirt.io/user-guide/virtual_machines/network_binding_plugins/#migration)
  section on how to define it.
- The backend `libvirt` and `qemu` needs to support migration for the specific
  device that is used in the domain.
  - The `hostdevice` devices in the domain do not support migration out-of-the-box.
    The network binding plugin infrastructure does not support migration for
    interfaces that have a `hostdevice` backend.

> **Note**:
> Network interfaces with `hostdevice` backend require a special handling from
> Kubevirt. One such handling is to unplug the device at the source before migration
> is started and plugging it back at the destination after the migration completes.
> Future development will consider adding a new migration method to support
> migration for such interfaces.

## Compute Resource Overhead

Some plugins may need additional resources to be added to the compute container of the virt-launcher pod.

It is possible to specify compute resource overhead that will be added to the `compute` container of virt-launcher pods
derived from virtual machines using the plugin.

> **Note**: At the moment, only memory overhead requests are supported.

For a network binding plugin to support compute resource overhead, the `computeResourceOverhead` field
must be specified in the Kubevirt CR.
See the user-guide network binding plugin [section](https://kubevirt.io/user-guide/network/network_binding_plugins/#register) on how to define it.

## Network plugin user sockets

Some plugins may need to create additional sockets beyond the gRPC one used for control communication between the sidecar and compute containers.
A common use case is establishing an extra communication channel between the compute container's virtualization stack and the network plugin running in the sidecar.

KubeVirt scans the `/var/run/kubevirt-hooks` directory in the sidecar for gRPC sockets to communicate with the sidecar.
Therefore, this directory must not contain any unrelated files or sockets.
To avoid conflicts, place additional sockets in a subdirectory under `/var/run/kubevirt-hooks`, such as `/var/run/kubevirt-hooks/userdata`.

Note: Some plugins may need to know the path accessible from the compute container for a specific sidecar.
In such case, use `/var/run/kubevirt-hooks/<sidecar container name>`. The sidecar's container name can be obtained from the `CONTAINER_NAME` environment variable.
