# Virtual Machine Configurations

## Motivation

So far KubeVirt provides a pod-like interface when managing VMIs. This seems to
be a good building block.
On the other hand this might not be sufficient for usual virtualization
management UIs, which do expect to manage the complete life-cycle of a VMI,
including shut down VMIs (which is currently not intended by the core runtime).


The purpose of this document is to describe a design that allows creating a VMI
configurations and how these can be used to create a VMI instances.  This
document specifically focuses on how the non-running and running VMI definitions
relate to each other.

Already known but not yet designed requirements were considered for this
design.


## Requirements
* Running and non-running VMIs are handled through the same API
* Creating a VMI must be separate from launching a VMI
* Support expanding sparse VMI definitions
* Support detailed VMI definitions


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
In addition to the existing `VMI` type, a new `VirtualMachineInstanceConfig` type is
getting introduced.
In contrast to the `VMI` object which represents a running VMI, the
`VirtualMachineInstanceConfig` object represents a static VMI configuration.
A `VirtualMachineInstanceConfig` object is used to create one or more `VMI` instances.

### Kinds

**VirtualMachineInstanceConfig**

```yaml
Kind: VirtualMachineInstanceConfig
Spec:
  Template:
    Metadata:
    Spec:
      Domain:
        Devices:
          …
```

**Instance**

The instance object `VMI` is equal to the already present entity in the core
KubeVirt runtime.

```yaml
Kind: VMI
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

The new `ownerReferences` field is pointing back to the `VirtualMachineInstanceConfig`
object which was used to create this `VMI` object.
It will be added to the metadata by the controller owning the
`VirtualMachineInstanceConfig` type.
The scheme can look similar to the `ownerReferences` field of a ReplicaSet
if it got created by a Deployment.

**Note:** ownerReferences must not be added or mutated by the user.


## Flow

The general concept is that the `VirtualMachineInstanceConfig` contains a static VMI
configuration.
Static refers to the fact that no runtime information is kept in such objects.

The runtime information of a VMI is kept in the `VMI` object, i.e. it's state
or computed values like bus addresses.

## Creation
To create a VMI definition, a user needs to `POST` a `VirtualMachineInstanceConfig`.

## Starting a VMI, the `/start` sub-resource
To create an instance from a `VirtualMachineInstanceConfig` a user must `GET` the `start`
sub-resource of a `VirtualMachineInstanceConfig` instance.

## Stopping a VMI, the `/stop` sub-resource
To stop a VMI, a user must `GET` the `stop` sub-resource of the associated
`VirtualMachineInstanceConfig` object.

**Note:** The semantics of a `DELETE` on the `VMI` object of a
`VirtualMachineInstanceConfig` still need to be defined.

## Changing a `VirtualMachineInstanceConfig`
Changes to a `VirtualMachineInstanceConfig` instance can be performed through the usual
`PATH` calls. Changing a `VirtualMachineInstanceConfig` does not lead to an automatic
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
2. A new VMI is created based on the config
   `GET` $c1/start
   → creates instance `POST` $i1
3. The VMI instance is getting stopped
   `GET` $c1/stop
   → `DELETE` $i1
4. The config is getting updated
   `PATCH` $c3
5. A new VMI instance is getting started
   `GET` $c1/start
   → creates instance `POST` $i2
6. The new VMI instance is getting stopped
   `GET` $c1/stop
   → deletes instance `DELETE` $i2

In this example, the user defines a `VirtualMachineInstanceConfig`, then starts it and
a `VMI` instance is created. Once the user stops it, the instance is removed.
After changing the `VirtualMachineInstanceConfig` (in step 4), he starts and stops the
VMI again.


### Review

Does this design meet the requirements?

* _Running and non-running VMIs are handled through the same API_ is given.
* _Creating a VMI must be separate from launching a VMI_ is also given by having
  the separate objects.
* _Support expanding sparse VMI definitions_ can happen at several stages.
* _Support detailed VMI definitions_ is also given, if needed a fully fledged
  definition can be specified in the VirtualMachineInstanceConfig


Does the design obviously prevent any other use-case we already aware of?

* _"Flavour", "Instance Type", or "Template" like functionality_ it looks like
  the static `VirtualMachineInstanceConfig` object provides enough flexibility.
* _Hot-(un-)plugging_ does not seem to be conflicting
* _Representation of dynamic instance specific data_ VMI instance is well suited
  for this.
* _Defaulting (to support OS compatibility)_ can also happen at multiple stages
* _Sub-resources for actions_ can be used
* _Presets to add certain configurations_ is also not conflicting with this
  design


## Implementation


The implementation for the current feature scope should be pretty straight
forward, as the flow is currently uni-directional, from `VirtualMachineInstanceConfig`
to `VMI`.
Thus a new simple controller can be used to handle this new type.
