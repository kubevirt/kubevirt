# Virtual Machine Configurations

## Motivation

So far KubeVirt provides a pod-like interface when managing VMs. This seems to
be a good building block.
On the other hand this might not be sufficient for usual virtualization
management UIs, which do expect to manage the complete life-cycle of a VM,
including shut down VMs (which is currently not intended by the core runtime).


The purpose of this document is to describe a design that allows creating a VM
configurations and how these can be used to create a VM instances.  This
document specifically focuses on how the non-running and running VM definitions
relate to each other.

Already known but not yet designed requirements were considered for this
design.


## Requirements
* Running and non-running VMs are handled through the same API
* Creating a VM must be separate from launching a VM
* Support expanding sparse VM definitions
* Support detailed VM definitions


## Additional context
In addition to the hard requirements, there are a few items which are likely
to become relevant in future, and should be considered in this design:

* "Flavour", "Instance Type", or "Template" like functionality
* Hot-(un-)plugging
* Representation of dynamic instance specific data
* Defaulting (to support OS compatibility)
* Sub-resources for actions
* Presets to add certain configurations

The design of these items is out of scope of this document.


## API
In addition to the existing `VM` type, a new `VirtualMachineConfig` type is
getting introduced.
In contrast to the `VM` object which represents a running VM, the
`VirtualMachineConfig` object represents a static VM configuration.
A `VirtualMachineConfig` object is used to create one or more `VM` instances.

### Kinds

**VirtualMachineConfig**

```yaml
Kind: VirtualMachineConfig
Spec:
  Template:
    Metadata:
    Spec:
      Domain:
        Devices:
          …
```

**Instance**

The instance object `VM` is equal to the already present entity in the core
KubeVirt runtime.

```yaml
Kind: VM
Metadata:
  ownerReferences: [{$config}]
Spec:
  Domain:
    Devices:
      …
State:
  Domain:
    …
```

The new `ownerReferences` field is pointing back to the `VirtualMachineConfig`
object which was used to create this `VM` object.
It will be added to the metadata by the controller owning the
`VirtualMachineConfig` type.
The scheme can look similar to the `ownerReferences` field of a ReplicaSet
if it got created by a Deployment.

**Note:** ownerReferences must not be added or mutated by the user.


## Flow

The general concept is that the `VirtualMachineConfig` contains a static VM
configuration.
Static refers to the fact that no runtime information is kept in such objects.

The runtime information of a VM is kept in the `VM` object, i.e. it's state
or computed values like bus addresses.

## Creation
To create a VM definition, a user needs to `POST` a `VirtualMachineConfig`.

## Starting a VM, the `/start` sub-resource
To create an instance from a `VirtualMachineConfig` a user must `GET` the `start`
sub-resource of a `VirtualMachineConfig` instance.

## Stopping a VM, the `/stop` sub-resource
To stop a VM, a user must `GET` the `stop` sub-resource of the associated
`VirtualMachineConfig` object.

**Note:** The semantics of a `DELETE` on the `VM` object of a
`VirtualMachineConfig` still need to be defined.

## Changing a `VirtualMachineConfig`
Changes to a `VirtualMachineConfig` instance can be performed through the usual
`PATH` calls. Changing a `VirtualMachineConfig` does not lead to an automatic
propagation of the changes to the instances.
However, in future there could be a specific metadata or action to trigger
such propagation automatically.


### Example Flow

Let's start with a simple example to get a better feeling for the relationship
between the objects.

```
$c1        $i1        $i2
POST
/start     POST
/stop      DELETE
PATCH
/start                POST
/stop                 DELETE
```

In other words:

1. A new config is created by a user
   `POST` $c1 config
2. A new VM is created based on the config
   `GET` $c1/start
   → creates instance `POST` $i1
3. The VM instance is getting stopped
   `GET` $c1/stop
   → `DELETE` $i1
4. The config is getting updated
   `PATCH` $c3
5. A new VM instance is getting started
   `GET` $c1/start
   → creates instance `POST` $i2
6. The new VM instance is getting stopped
   `GET` $c1/stop
   → deletes instance `DELETE` $i2

In this example, the user defines a `VirtualMachineConfig`, then starts it and
a `VM` instance is created. Once the user stops it, the instance is removed.
After changing the `VirtualMachineConfig` (in step 4), he starts and stops the
VM again.


### Review

Does this design meet the requirements?

* _Running and non-running VMs are handled through the same API_ is given.
* _Creating a VM must be separate from launching a VM_ is also given by having
  the separate objects.
* _Support expanding sparse VM definitions_ can happen at several stages.
* _Support detailed VM definitions_ is also given, if needed a fully fledged
  definition can be specified in the VirtualMachineConfig


Does the design obviously prevent any other use-case we already aware of?

* _"Flavour", "Instance Type", or "Template" like functionality_ it looks like
  the static `VirtualMachineConfig` object provides enough flexibility.
* _Hot-(un-)plugging_ does not seem to be conflicting
* _Representation of dynamic instance specific data_ VM instance is well suited
  for this.
* _Defaulting (to support OS compatibility)_ can also happen at multiple stages
* _Sub-resources for actions_ can be used
* _Presets to add certain configurations_ is also not conflicting with this
  design


## Implementation


The implementation for the current feature scope should be pretty straight
forward, as the flow is currently uni-directional, from `VirtualMachineConfig`
to `VM`.
Thus a new simple controller can be used to handle this new type.
