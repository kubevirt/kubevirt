Virtual Machine Presets
=============================

`VirtualMachineInstancePresets` are an extension to general `VirtualMachineInstance`
configuration behaving much like `PodPresets` from Kubernetes. When a
`VirtualMachineInstance` is created, any applicable `VirtualMachineInstancePresets`
will be applied to the existing spec for the `VirtualMachineInstance`. This allows
for re-use of common settings that should apply to multiple `VirtualMachineInstances`.


Implementation
------------------------

`VirtualMachineInstancePresets` are applied early while processing `VirtualMachineInstance`
resources. This means the `VirtualMachineInstance` resource is modified before it
is processed by any other component of KubeVirt.

Once a `VirtualMachineInstancePreset` is successfully applied to a `VirtualMachineInstance`,
the `VirtualMachineInstance` will be marked with an annotation to indicate that it
was applied. If a conflict occurs while a `VirtualMachineInstancePreset` is being
applied that portion of the `VirtualMachineInstancePreset` will be skipped.


Usage
------------------------

KubeVirt uses Kubernetes `Labels` and `Selectors` to determine which
`VirtualMachineInstancePresets` apply to any given `VirtualMachineInstance`, similarly to how
`PodPresets` work in Kubernetes. The `VirtualMachineInstance` is marked with an
Annotation upon successful completion.

Any domain structure can be listed in the `spec` of a `VirtualMachineInstancePreset`.
e.g. Clock, Features, Memory, CPU, or Devices such network interfaces.  All
elements of the `spec` section of a `VirtualMachineInstancePreset` will be applied
to the `VirtualMachineInstance`.


Overrides
------------------------

`VirtualMachineInstancePresets` use a similar conflict resolution strategy to
Kubernetes `PodPresets`. If a portion of the domain spec is present in both a
`VirtualMachineInstance` and a `VirtualMachineInstancePreset` and both resources have the
identical information, then creation of the `VirtualMachineInstance` will continue
normally. If however there is a difference between the resources, an Event will
be created indicating which `DomainSpec` element of which `VirtualMachineInstancePreset`
was overridden. For example: If both the `VirtualMachineInstance` and
`VirtualMachineInstancePreset` define a `CPU`, but use a different number of `Cores`,
KubeVirt will note the difference.

If any settings from the `VirtualMachineInstancePreset` were successfully applied, the
`VirtualMachineInstance` will be annotated.

Because `VirtualMachineInstancePresets` are implemented within the `virt-controller` pod,
log messages associated with resource conflicts will also be reflected there.


Creation and Usage
------------------------

`VirtualMachineInstancePresets` are namespaced resources, so should be created in the
same namespace as the `VirtualMachineInstances` that will use them:

`kubectl create -f <preset>.yaml [--namespace <namespace>]`

KubeVirt will determine which `VirtualMachineInstancePresets` apply to a Particular
`VirtualMachineInstance` by matching `Labels`. For example:

```yaml
kind: VirtualMachineInstancePreset
metadata:
  name: example-preset
spec:
  selector:
    matchLabels:
      kubevirt.io/flavor: foo
  ...
```

would match any `VirtualMachineInstance` in the same namespace with a `Label` of
`kubevirt.io/flavor: foo`. For example:

```yaml
kind: VirtualMachineInstance
version: v1
metadata:
  name: myvm
  labels:
    kubevirt.io/flavor: foo
  ...
```


Exclusions
------------------------
Since `VirtualMachineInstancePresets` use `Selectors` that indicate which
`VirtualMachineInstances` their settings should apply to, there needs to exist a
mechanism by which `VirtualMachineInstances` can opt out of `VirtualMachineInstancePresets`
altogether. This is done using an annotation:

```yaml
kind: VirtualMachineInstance
version: v1
metadata:
  name: myvm
  annotations:
    virtualmachinepresets.admission.kubevirt.io/exclude: "true"
  ...
```


Examples
=============================

Simple `VirtualMachineInstancePreset` Example
------------------------

```yaml
apiVersion: kubevirt.io/v1alpha2
kind: VirtualMachineInstancePreset
metadata:
  name: example-preset
spec:
  selector:
    matchLabels:
      kubevirt.io/flavor: windows-10
  domain:
    features:
      acpi: {}
      apic: {}
      hyperv:
        relaxed: {}
        vapic: {}
        spinlocks:
          spinlocks: 8191
---
apiVersion: kubevirt.io/v1alpha2
kind: VirtualMachineInstance
version: v1
metadata:
  name: myvm
  labels:
    kubevirt.io/flavor: windows-10
spec:
  domain:
    firmware:
      uuid: c8f99fc8-20f5-46c4-85e5-2b841c547cef
```

Once the `VirtualMachineInstancePreset` is applied to the `VirtualMachineInstance`, the
resulting resource would look like this:


```yaml
apiVersion: v1
items:
- apiVersion: kubevirt.io/v1alpha2
  kind: VirtualMachineInstance
  metadata:
    annotations:
      presets.virtualmachines.kubevirt.io/presets-applied: kubevirt.io/v1alpha2
      virtualmachinepreset.kubevirt.io/example-preset: kubevirt.io/v1alpha2
    labels:
      kubevirt.io/flavor: windows-10
    name: myvm
    namespace: default
    selfLink: /apis/kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances/myvm
  spec:
    domain:
      devices: {}
      features:
        acpi:
          enabled: true
        apic:
          enabled: true
        hyperv:
          relaxed:
            enabled: true
          spinlocks:
            enabled: true
            spinlocks: 8191
          vapic:
            enabled: true
      firmware:
        uuid: c8f99fc8-20f5-46c4-85e5-2b841c547cef
      machine:
        type: q35
      resources:
        requests:
          memory: 8Mi
  status:
    phase: Scheduling
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

Conflict Example
------------------------

This is an example of a merge conflict. In this case both the `VirtualMachineInstance`
and `VirtualMachineInstancePreset` request different number of CPU's.


```yaml
apiVersion: kubevirt.io/v1alpha2
kind: VirtualMachineInstancePreset
metadata:
  name: example-preset
spec:
  selector:
    matchLabels:
      kubevirt.io/flavor: default-features
  domain:
    cpu:
      cores: 4
---
apiVersion: kubevirt.io/v1alpha2
kind: VirtualMachineInstance
version: v1
metadata:
  name: myvm
  labels:
    kubevirt.io/flavor: default-features
spec:
  domain:
    cpu:
      cores: 6
```

In this case the `VirtualMachineInstance` Spec will remain unmodified. Use
`kubectl get events` to show events.

```yaml
apiVersion: v1
items:
- apiVersion: kubevirt.io/v1alpha2
  kind: VirtualMachineInstance
  metadata:
    annotations:
      presets.virtualmachines.kubevirt.io/presets-applied: kubevirt.io/v1alpha2
    clusterName: ""
    labels:
      kubevirt.io/flavor: default-features
    name: myvm
    namespace: default
    selfLink: /apis/kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances/myvm
  spec:
    domain:
      cpu:
        cores: 6
      devices: {}
      features:
        acpi:
          enabled: true
      firmware:
        uuid: efaaa6e4-0002-44d6-9de1-5526b24615d1
      machine:
        type: q35
      resources:
        requests:
          memory: 8Mi
  status:
    phase: Scheduling
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

Calling `kubectl get events` would have a line like:
2m          2m           1         myvm.1515bbb8d397f258                       VirtualMachineInstance                                     Warning   Conflict                  virtualmachine-preset-controller   Unable to apply VirtualMachineInstancePreset 'example-preset': spec.cpu: &{6} != &{4}
