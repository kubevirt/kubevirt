# KubeVirt Glossary

This document exists to to give the KubeVirt community a common language to
work with when describing features and making design proposals. Some of the
terms in this document do not actually exist in the current design yet, but
are instead concepts the KubeVirt team is in the process of formalizing. As new
concepts are introduced into the community, this document should be updated to
reflect the continued evolution of our common language. 

Terms for Items the KubeVirt community are still discussing are marked with the
(WIP) tag.

## VM Definition

A VM Definition is the declaration of a desired state of a VM.

Below is an example of a VM definition. A user can post this definition to the
cluster and expect the KubeVirt runtime to fulfill creating a VM matching the
posted definition.
```
kind: VirtualMachine
metadata:
  name: testvm
spec:
  domain:
    devices:
      disks:
      - type: network
        snapshot: external
        device: disk
        driver:
          name: qemu
          type: raw
          cache: none
        source:
          host:
            name: iscsi-demo-target.default
            port: "3260"
          protocol: iscsi
          name: iqn.2017-01.io.kubevirt:sn.42/2
        target:
          dev: vda
    memory:
      unit: MB
      value: 64
    os:
      type:
        os: hvm
    type: qemu

```

## VM Specification aka VM Spec

The VM Specification is the part of the VM Definition found in the **spec**
field. The VM spec mostly contains information about devices and disks
associated with the VM.

At VM creation time, the VM spec is transformed into Domain XML and handed off
to libvirt.

## VM Status

In contrast to the VM Definition, the VM Status expresses the actual state of
the VM. The status is visible when querying an active VM from the KubeVirt
runtime.

## VM Specification Completion (WIP)

This term represents the set of actions that are performed on a VM’s Spec
during the VM Creation Process. This includes injecting presets, libvirt
defaulting, but is not limited to only those items. Basically all kinds of VM
manipulation and validation which happen when a VM is posted is covered by that
(admission controllers, initializers, external admission controllers). After
this completion phase is done, a fully initialized and complete VM spec is
generated and persisted.

## VirtualMachine aka VM

A VM is a cluster object representation of a single emulated computer system
running on a cluster node. A **VM Definition** is a declaration of a desired
state of a Virtual Machine. When such a definition is posted to the KubeVirt
runtime, you want to see the described Virtual Machine running. The VM object
maps to exactly one actively running Virtual Machine instance, because of this
the VM object should be used to represent any active Virtual Machineinstance’s
specific runtime data.

A VM Definition is posted and the KubeVirt runtime creates the VM object to
represent the active VM process running on a cluster node.

## VirtualMachineConfig aka VMC (WIP)

Posting a VM definition causes the KubeVirt runtime to immediately create a VM
instance in response to the POST request. When the VM definition is deleted,
this results in the VM being stopped.

A VirtualMachineConfig is a concept that allows users to post a representation
of a VM into the cluster in a way that is de-coupled from start/stopping the VM.

A user can post a VirtualMachineConfig and later choose to start a VM using that
config.  This lets the config remain persistent between VM starts/stops.

## VirtualMachineGroup aka VMG (WIP)

A VirtualMachineGroup is a concept that allows users to post a representation of
a VM along with the desired number of cloned VM instances to run using that
representation.  The KubeVirt runtime will manage starting and stopping
instances to match the desired number of cloned instances defined by the VMG.

## KubeVirt Cluster Node

The underlying hardware KubeVirt is scheduling VMs on top of. Any node KubeVirt
is capable of starting a VM on is considered a cluster node.

## KubeVirt Runtime

The core software components that make up KubeVirt. Runtime related code lives
in kubevirt/kubevirt. It provides controllers, specifications and definitions,
which allow users to express desired VirtualMachine states, and will try to
fulfill them. If definitions of VMs are made known to the runtime, the runtime
will immediately try to fulfill the request by instantiating and starting that
VM.

## VirtualMachinePreset aka VMP (WIP)

Likely, like a PodPreset. It will allow to inject specific resources into a VM.
VMP is a good candidate for injecting disks and devices into VMs during VM spec
completion.

## RegistryDisk (WIP)

Method of storing and distributing VM disks with KubeVirt using the container
registry.

## Domain
Libvirt domain. `virt-handler` can derive a Domain XML out of a [VM Spec](#vm-specification-aka-vm-spec).
This is the host centric view of the cluster wide [VM Spec](#vm-specification-aka-vm-spec).

## Domain XML

Configuration used to define a domain in Libvirt.  The VM spec is transformed
into domain xml during VM creation on a cluster node. The Domain xml is used to
communicate with Libvirt the information pertaining to how the VM should be
launched.

## Third Party Resource aka TPR
Kubernetes has an extensible API which allows extending its REST-API.
Resources using this extension mechanism are called Third Party Resource.
See [extensible-api](https://github.com/kubernetes/kubernetes/blob/master/docs/design/extending-api.md)
for more information.


## Admission Controller (WIP)

Plug-ins built into Kubernetes, which intercept requests after authentication.
They can alter or reject resources before they are persisted.

https://kubernetes.io/docs/admin/admission-controllers/#what-are-they

## External Webhook Admission Controller (WIP)

Like admission controllers, but the admission controller logic can live outside
of kubernetes and is invoked via a webhook.

https://kubernetes.io/docs/admin/extensible-admission-controllers/#what-are-external-admission-webhooks

## Initializer (WIP)

Another way of mutating or validating objects before a request is finished.
However, in contrast to admission controllers, initializers can perform their
logic asynchronously. First objects are marked as having the need to be
initialized by one or more initializers. An external controller can listen
on the apiserver for these uninitialized objects and try to initialize them
The advantage here is, that you can write traditional controllers with
workqueues, since the object is already persisted.

However, for the user POSTs are blocked until the object is fully initialized
and other normal requests will not show the object until it is ready. So the
fact that this works asynchronous is hidden from the user.

https://kubernetes.io/docs/admin/extensible-admission-controllers/#what-are-initializers
