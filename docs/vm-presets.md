Virtual Machine Presets
=============================

`VirtualMachinePresets` are an extension to general `VirtualMachine`
configuration behaving much like `PodPresets` from Kubernetes. When a
`VirtualMachine` is created, any applicable `VirtualMachinePresets`
will be applied to the existing spec for the `VirtualMachine`. This allows
for re-use of common settings that should apply to multiple `VirtualMachines`.


Implementation
------------------------

`VirtualMachinePresets` are implemented as a Kubernetes `Initializer`. This
means the `VirtualMachine` resource is modified before it is visible to or
processed by any other component of KubeVirt.

Once a `VirtualMachinePreset` is successfully applied to a `VirtualMachine`,
the `VirtualMachine` will be marked with an annotation to indicate that it
was applied. If an error occurs while a `VirtualMachinePreset` is being applied
(for example, if a conflict occurs), none of the `VirtualMachinePreset` will be
applied.


Usage
------------------------

KubeVirt uses Kubernetes `Labels` and `Selectors` to determine which
`VirtualMachinePresets` apply to any given `VirtualMachine`, similarly to how
`PodPresets` work in Kubernetes. The `VirtualMachine` is marked with an
Annotation upon successful completion.

Any domain structure can be listed in the `spec` of a `VirtualMachinePreset`.
e.g. Clock, Features, Memory, CPU, or Devices such network interfaces or disks.
All elements of the `spec` section of a `VirtualMachinePreset` will be applied
to the `VirtualMachine`.


Conflicts
------------------------

`VirtualMachinePresets` use a similar conflict resolution strategy to
Kubernetes `PodPresets`. If a portion of the domain spec is present in both a
`VirtualMachine` and a `VirtualMachinePreset` and both resources have the
identical information, then no error will occur and `VirtualMachine` creation
will continue normally. If however there is a conflict between the resources,
an error will occur and the `VirtualMachine` will not be created. For example:
If both the `VirtualMachine` and `VirtualMachinePreset` define a `Volume`, but
use different paths for the same name, KubeVirt will note the conflict.

Because `VirtualMachinePresets` are implemented as an Initializer within the
`virt-controller` pod, log messages associated with resource conflicts will
also be reflected there.


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
  selector:
    matchLabels:
      flavor: foo
  ...
```

would match any `VirtualMachine` in the same namespace with a `Label` of
`flavor: foo`. For example:

```yaml
kind: VirtualMachine
version: v1
metadata:
  name: myvm
  labels:
    flavor: foo
  ...
```


Examples
=============================

Simple `VirtualMachinePreset` Example
------------------------

```yaml
kind: VirtualMachinePreset
version: v1alpha1
metadata:
  name: example-preset
  selector:
    matchLabels:
      flavor: default-features
spec:
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
kind: VirtualMachine
version: v1
metadata:
  name: myvm
  labels:
    flavor: default-features
spec:
  firmware:
    UUID: c8f99fc8-20f5-46c4-85e5-2b841c547cef
```

Once the `VirtualMachinePreset` is applied to the `VirtualMachine`, the
resulting resource would look like this:


```yaml
kind: VirtualMachine
version: v1
metadata:
  name: myvm
  labels:
    flavor: windows-10
  annotations:
    virtualmachinepreset.kubevirt.io/example-preset: kubevirt.io/v1alpha1
spec:
  firmware:
    UUID: c8f99fc8-20f5-46c4-85e5-2b841c547cef
  domain:
    features:
      acpi: {}
      apic: {}
      hyperv:
        relaxed: {}
        vapic: {}
        spinlocks:
          spinlocks: 8191
```


Merging Resources Example
------------------------

Here's an example with multiple volumes to demonstrate merging of devices.

```yaml
kind: VirtualMachinePreset
version: v1alpha1
metadata:
  name: windows-features
  selector:
    matchLabels:
      flavor: default-features
spec:
  domain:
    disks:
    - name: server2012r2
      volumeName: server2012r2
      disk:
        dev: vda
  volumes:
    - name: server2012r2
      iscsi:
        iqn: iqn.2018-01.io.kubevirt:sn.42
        lun: 4
        targetPortal: iscsi-demo-target.kube-system.svc.cluster.local
---
kind: VirtualMachine
version: v1
metadata:
  name: myvm
  labels:
    flavor: default-features
spec:
  domain:
    disks:
    - name: varlog
      volumeName: varlog
      disk:
        dev: vdb
  volumes:
    - name: varlog
      iscsi:
        iqn: iqn.2018-02.io.kubevirt:sn.42
        lun: 5
        targetPortal: iscsi-demo-target.kube-system.svc.cluster.local
```


```yaml
kind: VirtualMachine
version: v1
metadata:
  name: myvm
  labels:
    flavor: windows-server2012r2
  annotations:
    virtualmachinepreset.kubevirt.io/windows-features: kubevirt.io/v1alpha1
spec:
  domain:
    disks:
    - name: varlog
      volumeName: varlog
      disk:
        dev: vdb
    - name: server2012r2
      volumeName: server2012r2
      disk:
        dev: vda
  volumes:
    - name: varlog
      iscsi:
        iqn: iqn.2018-02.io.kubevirt:sn.42
        lun: 5
        targetPortal: iscsi-demo-target.kube-system.svc.cluster.local
    - name: server2012r2
      iscsi:
        iqn: iqn.2018-01.io.kubevirt:sn.42
        lun: 4
        targetPortal: iscsi-demo-target.kube-system.svc.cluster.local
```


Conflict Example
------------------------

This is an example of a merge conflict. In this case the disk and volume are
nearly identical, but the specified disks have a conflcting device node.


```yaml
kind: VirtualMachinePreset
version: v1alpha1
metadata:
  name: example-preset
  selector:
    matchLabels:
      flavor: default-features
spec:
  domain:
    disks:
    - name: server2012r2
      volumeName: server2012r2
      disk:
        dev: vda
  volumes:
    - name: server2012r2
      iscsi:
        iqn: iqn.2018-01.io.kubevirt:sn.42
        lun: 4
        targetPortal: iscsi-demo-target.kube-system.svc.cluster.local
---
kind: VirtualMachine
version: v1
metadata:
  name: myvm
  labels:
    flavor: default-features
spec:
  domain:
    disks:
    - name: server2012r2
      volumeName: server2012r2
      disk:
        dev: vdb
  volumes:
    - name: server2012r2
      iscsi:
        iqn: iqn.2018-01.io.kubevirt:sn.42
        lun: 4
        targetPortal: iscsi-demo-target.kube-system.svc.cluster.local
```

In this case the `VirtualMachine` will remain unmodified. Use
`kubectl describe` to show events.
