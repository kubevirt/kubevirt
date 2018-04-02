Virtual Machine Presets
=============================

`VirtualMachinePresets` are an extension to general `VirtualMachine`
configuration behaving much like `PodPresets` from Kubernetes. When a
`VirtualMachine` is created, any applicable `VirtualMachinePresets`
will be applied to the existing spec for the `VirtualMachine`. This allows
for re-use of common settings that should apply to multiple `VirtualMachines`.


Implementation
------------------------

`VirtualMachinePresets` are applied early while processing `VirtualMachine`
resources. This means the `VirtualMachine` resource is modified before it
is processed by any other component of KubeVirt.

Once a `VirtualMachinePreset` is successfully applied to a `VirtualMachine`,
the `VirtualMachine` will be marked with an annotation to indicate that it
was applied. If a conflict occurs while a `VirtualMachinePreset` is being
applied that portion of the `VirtualMachinePreset` will be skipped.


Usage
------------------------

KubeVirt uses Kubernetes `Labels` and `Selectors` to determine which
`VirtualMachinePresets` apply to any given `VirtualMachine`, similarly to how
`PodPresets` work in Kubernetes. The `VirtualMachine` is marked with an
Annotation upon successful completion.

Any domain structure can be listed in the `spec` of a `VirtualMachinePreset`.
e.g. Clock, Features, Memory, CPU, or Devices such network interfaces.  All
elements of the `spec` section of a `VirtualMachinePreset` will be applied
to the `VirtualMachine`.


Overrides
------------------------

`VirtualMachinePresets` use a similar conflict resolution strategy to
Kubernetes `PodPresets`. If a portion of the domain spec is present in both a
`VirtualMachine` and a `VirtualMachinePreset` and both resources have the
identical information, then creation of the `VirtualMachine` will continue
normally. If however there is a difference between the resources, an Event will
be created indicating which `DomainSpec` element of which `VirtualMachinePreset`
was overridden. For example: If both the `VirtualMachine` and
`VirtualMachinePreset` define a `CPU`, but use a different number of `Cores`,
KubeVirt will note the difference.

If any settings from the `VirtualMachinePreset` were successfully applied, the
`VirtualMachine` will be annotated.

Because `VirtualMachinePresets` are implemented within the `virt-controller` pod,
log messages associated with resource conflicts will also be reflected there.


Creation and Usage
------------------------

`VirtualMachinePresets` are namespaced resources, so should be created in the
same namespace as the `VirtualMachines` that will use them:

`kubectl create -f <preset>.yaml [--namespace <namespace>]`

KubeVirt will determine which `VirtualMachinePresets` apply to a Particular
`VirtualMachine` by matching `Labels`. For example:

```yaml
kind: VirtualMachinePreset
metadata:
  name: example-preset
spec:
  selector:
    matchLabels:
      kubevirt.io/flavor: foo
  ...
```

would match any `VirtualMachine` in the same namespace with a `Label` of
`kubevirt.io/flavor: foo`. For example:

```yaml
kind: VirtualMachine
version: v1
metadata:
  name: myvm
  labels:
    kubevirt.io/flavor: foo
  ...
```


Examples
=============================

Simple `VirtualMachinePreset` Example
------------------------

```yaml
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachinePreset
version: v1alpha1
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
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachine
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

Once the `VirtualMachinePreset` is applied to the `VirtualMachine`, the
resulting resource would look like this:


```yaml
apiVersion: v1
items:
- apiVersion: kubevirt.io/v1alpha1
  kind: VirtualMachine
  metadata:
    annotations:
      presets.virtualmachines.kubevirt.io/presets-applied: kubevirt.io/v1alpha1
      virtualmachinepreset.kubevirt.io/example-preset: kubevirt.io/v1alpha1
    labels:
      kubevirt.io/flavor: windows-10
    name: myvm
    namespace: default
    selfLink: /apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/myvm
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

This is an example of a merge conflict. In this case both the `VirtualMachine`
and `VirtualMachinePreset` request different number of CPU's.


```yaml
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachinePreset
version: v1alpha1
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
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachine
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

In this case the `VirtualMachine` Spec will remain unmodified. Use
`kubectl get events` to show events.

```yaml
apiVersion: v1
items:
- apiVersion: kubevirt.io/v1alpha1
  kind: VirtualMachine
  metadata:
    annotations:
      presets.virtualmachines.kubevirt.io/presets-applied: kubevirt.io/v1alpha1
    clusterName: ""
    labels:
      kubevirt.io/flavor: default-features
    name: myvm
    namespace: default
    selfLink: /apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/myvm
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
2m          2m           1         myvm.1515bbb8d397f258                       VirtualMachine                                     Warning   Conflict                  virtualmachine-preset-controller   Unable to apply VirtualMachinePreset 'example-preset': spec.cpu: &{6} != &{4}
