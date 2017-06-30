# VM Configuration creation

## Motivation

TBD define use cases

KubeVirt should allows users to start small, but also provide fully fledged
VM definitions.
In that sense a user should be able to just name a disk and for example the
operating system which is provided by that disk, and will get an optimized VM
instance for this specific combination.

This document is focused on differentiating the different mechanics and how
they play together to support the _start small_ (or sparse VM definition)
use-case.

This topic will be visited from two sides:
1. Compose - How the user can put together a VM definition
2. Complete - How the system transforms this potentially partial definition
   into a useful one


## Requirements
There are certain requirements which should be met 

* Allow creation of building blocks
* Allow setting defaults for certain expansions/inferences
* Allow capturing the output of Config post-processing


## Compose

### User Config

The user starts with writing a minimal VM configuration.

An example:

```yaml
kind: Config
metadata:
  labels:
    distro: suse-10
    instanceType: small
spec:
  domain:
    devices:
      disk:
        claimName: my-suse
```

This definition is completly user driven.


### Presets

The first way how this user provided config can be extended is by using
presets.
Conceptualy presets work like Kubernetes Pod
[Presets](https://kubernetes.io/docs/tasks/inject-data-application/podpreset/).
A preset is defined, and applied to a `Config` based on a selector.

For example, the admin could have defined a preset to apply certain defaults
if a `Config` is labeled with `distro: suse-10`, such a preset could look like
the following snippet:

```yaml
Kind: VMPreset
metadata:
  name: SuSE_10_Defaults
spec:
  selector:
    matchLabels:
      distro: suse-10
  domain:
    default:
      devices:
        disk:
          model: virtio
        interfaces:
          model: virtio
```

This preset tells the system to merge the `.spec.domain` part of the preset,
with each `Config` wich matches the label `distro: suse-10`.
In this example the preset would set the default for certain devices.

The following example uses a preset to set certain memory and vcpu values to
achieve something like instance type functionality:

```yaml
Kind: VMPreset
metadata:
  name: SMALL_Instance
spec:
  selector:
    matchLabels:
      instanceType: small
  domain:
    memory:
      value: 512
    vcpu:
      value: 2
```

This would set (and override) the memory to 512MB and the number of CPU cores
to 2 for all `Config`s which are labeled with `instanceType: small`.

The example above would overwrite any user provided memory and CPU values.
If this should not happen, then the same preset could use defaults, to just
set these values in case that they are not provided:

```yaml
Kind: VMPreset
metadata:
  name: SMALL_Instance
spec:
  selector:
    matchLabels:
      instanceType: small
  domain:
    default:
      memory:
        value: 512
      vcpu:
        value: 2
```

In this case, the values would just be set if the user did not provide them.

## Complete
Up to now the config was extended on pure syntactical levels.
In the following steps, semantical and syntactical changes are performed.


### Expansion or dependency inference
This step is about adding required device dependencies to a VM configuration
to enable the given device.

For example, if a user adds a disk to a VM configuration, then he does not
necessarily also add a relevant controller. but this controller is needed to
connect the disk with the virtual machine.

This process is called expansion.


### Defaulting
Whenever a device needs to be added during the expansion process, then the
system needs to decide what specific device should be used.

For example, if a controller needs to be added to connect a disk, then the
system needs to decide what kind of controller to add (USB, IDE, virtio, …).

The type of controller is relevant, because operating systems typically just
support a certain subset of the available busses.

This decision on a default value is called defaulting.
