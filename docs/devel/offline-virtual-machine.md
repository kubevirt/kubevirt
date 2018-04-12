# Offline Virtual Machine developer documentation

This document introduces the OfflineVirtualMachine kind and provides a
guide how to use it and build upon it.

## What is Offline Virtual Machine

Almost all virtual machine (VM) management systems allow you to manage both running
and stopped virtual machines. Such system allows you to edit configuration of
both types of VMs and show its statuses.

To allow building such VM management systems on top of the KubeVirt, the
OfflineVirtualMachine is introduced to provide the access to the stopped
virtual machines. When working with running virtual machines, please see
the [VirtualMachine] object documentation. The Virtual Machine object is
designed to work in tandem with the OfflineVirtualMachine and provides the
configuration and status for running virtual machines.

OfflineVirtualMachine is a Kubernetes [custom resource definition](https://kubernetes.io/docs/concepts/api-extension/custom-resources/), which
allows for using the Kubernetes machanisms for storing the objects and
exposing it through the API.

## What it does and how to use it

The OfflineVirtualMachine provides the functionality to:

* Store OfflineVirtualMachine,
* Manipulate the OfflineVirtualMachine through the kubectl,
* Manipulate the OfflineVirtualMachine through the Kubernetes API,
* Watch for changes in the OfflineVirtualMachine and react to them:
  * Convert the OfflineVirtualMachine to VirtualMachine and thus launch it
  * Stop VirtualMachine and update status of OfflineVirtualMachine accordingly

### Kubectl interface

The kubectl allows you to manipulate the Kubernetes objects imperatively.
You can create, delete, update and query objects in the API. More details on
how to use kubectl and what it can do are in the [Kubernetes documentation](https://kubernetes.io/docs/reference/kubectl/overview/).

Following are the examples of working with OfflineVirtualMachine and kubectl:

```bash
# Define an OfflineVirtualMachine:
kubectl create -f myofflinevm.yaml

# Start an OfflineVirtualMachine:
kubectl patch offlinevirtualmachine myvm --type=merge -p \
    '{"spec":{"running": true}}'

# Look at OfflineVirtualMachine status and associated events:
kubectl describe offlinevirtualmachine myvm

# Look at the now created VirtualMachine status and associated events:
kubectl describe virtualmachine myvm

# Stop an OfflineVirtualMachine:
kubectl patch offlinevirtualmachine myvm --type=merge -p \
    '{"spec":{"running": false}}'

# Implicit cascade delete (first deletes the vm and then the ovm)
kubectl delete offlinevirtualmachine myvm

# Explicit cascade delete (first deletes the vm and then the ovm)
kubectl delete offlinevirtualmachine myvm --cascade=true

# Orphan delete (The running vm is only detached, not deleted)
# Recreating the ovm would lead to the adoption of the vm
kubectl delete offlinevirtualmachine myvm --cascade=false
```

### The REST API

The kubectl is a handy tool that provides handy access to cluster, when you sit
at the console. But, when you are writting an external application that
needs to access the cluster programaticaly, it is better to have a API endpoint.
Thats where the Kubernetes REST API endpoint comes right in. Kubernetes provides
for its users the native REST API, which is easily extendable and in one place.

The OfflineVirtualMachine object is a CRD, which implies that Kubernetes
provides the API automatically. The API is located at the path

```text
<your-api-server-adress>/apis/kubevirt.io/v1alpha/offlinevirtualmachine/
```

and you can do typical REST manipulation, you would expect. All CRUD is
supported, as shown in following example.

```text
POST /apis/kubevirt.io/v1alpha1/namespaces/{namespace}/offlinevirtualmachine
GET /apis/kubevirt.io/v1alpha1/namespaces/{namespace}/offlinevirtualmachine
GET /apis/kubevirt.io/v1alpha1/namespaces/{namespace}/offlinevirtualmachine/{name}
DELETE /apis/kubevirt.io/v1alpha1/namespaces/{namespace}/offlinevirtualmachine/{name}
PUT /apis/kubevirt.io/v1alpha1/namespaces/{namespace}/offlinevirtualmachine/{name}
PATCH /apis/kubevirt.io/v1alpha1/namespaces/{namespace}/offlinevirtualmachine/{name}
```

By **POST** you can store new object in the etcd. With **GET** you either
get the list of all OfflineVirtualMachines or get the concrete one. **DELETE**
removes the object from etcd and all its resources. If you want to update the
existing OfflineVirtualMachine object use **PUT** and if you want to change
an item inside the object use **PATCH**.
More details on the API are in the [documentation](https://kubevirt.github.io/api-reference/master/operations.html).

To data format used when communicating with the API is the JSON. The format is
set up the usual way by setting the Content-Type header to 'application/json'.
The 'application/yaml' can also be used.

## OfflineVirtualMachine object content

Now its time to discuss the content of the OfflineVirtualMachine object.
The object is defined in the same way as any other Kubernetes object. You can
use YAML or JSON format to specify the content. The example structure is bellow:

```yaml
apiVersion: kubevirt.io/v1alpha1
kind: OfflineVirtualMachine
metadata:
  name: myvm
spec:
  running: false
  template:
    metadata:
      labels:
        my: label
    spec:
      domain:
        resources:
          requests:
            memory: 8Mi
        devices:
          disks:
          - name: disk0
            volumeName: mypcv
      volumes:
        - name: mypvc
          persistentVolumeClaim:
            claimName: myclaim
```

The file specification follows the Kubernetes guide. The apiVersion is linked
with the Kubevirt release cycle.

In the metadata section, there is a *required* field, the **name**. Then
following the spec section, there are two important parts. The **running**, which
indicates the current state of the VirtualMachine attached to this VirtualMachine.
Second is the **template**, which is the VirtualMachine template.

Let us go over each of these fields.

### Name

The `metadata.name` is important for the OfflineVirtualMachine, it is used to find
created VirtualMachine. The VirtualMachine name is directly derived from
the OfflineVirtualMachine. This means, if OfflineVirtualMachine is names 'testvm',
the VirtualMachine is named 'testvm' too.

Moreover, the pair namespace and name: `namepace/metadata.name` creates the
unique identifier for the OfflineVirtualMachine. This fact implies that two
identical names in the same namespace are considered as an error.

### Template

The spec.template is a VirtualMachine specification used to create an actual
VirtualMachine. Below is an example of such VirtualMachine specification.

```yaml
metadata:
spec:
  domain:
    resources:
      requests:
        memory: 8Mi
    devices:
      disks:
      - name: disk0
        volumeName: mypcv
  volumes:
    - name: mypvc
      persistentVolumeClaim:
        claimName: myclaim
```

It is easy to see that it is exactly the same as [VirtualMachine],
but it does not have a `kind` and `apiVersion`. These are implicitely added.

Thus for the complete list of supported fields in the spec.template please refer
to the [VirtualMachine] documentation.

## OfflineVirtualMachine behaviour

The OfflineVirtualMachine has to be in sync with its VirtualMachine. This means
that the OfflineVirtualMachine controller has to observe both, the OfflineVirtualMachine
and created VirtualMachine. When the [link](#ownerreference) is established the config changes
are translated to the VirtualMachine, and coresponding status changes are
translated back to OfflineVirtualMachine.

TBD this needs to be more specific

### Status

Shows the information about current OfflineVirtualMachine. The example status
is shown below.

```yaml
status:
  observedGeneration: 124 # current observed revision
  virtualMachine: my-vm
  running: true # is the attached VirtualMachine running
  ready: true # based on http readiness check libvirt info
  conditions: [] # additional possible states
```

The status of the VirtualMachine is watched and is reflected in the
OfflineVirtualMachine status. The info propagated from the VirtualMachine is:

* running state
* readiness of the VM
* name of the VirtualMachine

For more information, user have to check the VirtualMachine object itself.

The conditions can show additional information about state of VirtualMachine.
One possible case would be to show how VirtualMachine was stopped, e.g.:

```yaml
status:
  conditions:
    - lastShutdown: 12.12.2018 00:00:00
      Reason: error
```

### OwnerReference

Linking the created VirtualMachine to its parent OfflineVirtualMachine pose
a challenge. Using the same name is only part of the solution. To find the
parent OfflineVirtualMachine programatically in the Kubernetes, the OwnerReference
is used. As described in the
[design document](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/controller-ref.md),
the OwnerReference lives in the metadata section of the object and is created
automaticaly. Example:

```YAML
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachine
metadata:
  name: myvm
  ownerReferences:
    - controller: true
      uid: 1234-1234-1234-1234
      kind: OfflineVirtualMachine
      version: kubevirt.io/v1alpha1
```

### Update strategy

For now implicit: OnDelete. Can later be extended to RollingUpdate if needed.
Spec changes have no direct effect on already running VMs, and they will not
directly be propagated to the VM. If a VM should be running (spec.running=true)
and it is powered down (VM object delete, OS shutdown, ...),
the VM will be re-created by the controller with the new spec.

### Delete strategy

The delete has a cascade that deletes the created VirtualMachine. If a cascade
is turned off the VirtualMachine is orphaned and leaved running.
When the OfflineVirtualMachine with the same name as orphaned VirtualMachine
is created, the VirtualMachine gets adopted and OwnerReference
is updated accordingly.

### Reset and Reboot

This is not covered by the OfflineVirtualMachine. This functionality shall
be achieved by subresources for the VirtualMachine (imperative),
and will not result in a recreation of the VirtualMachine object or its Pod.
From the KubeVirt perspective, the VirtualMachine is running all the time.
Thus spec changes on OfflineVirtualMachine will not be propagated to
the VirtualMachine in that case.

## How it is implemented

The first implementation of the OfflineVirtualMachine kind has two parts:

1) The OfflineVirtualMachine custom resource definition,
2) The controller watching for the changes and updating the state.

### Custom resource definition

The OfflineVirtualMachine custom resource is straightforward and is shown bellow:

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: offlinevirtualmachines.kubevirt.io
spec:
  scope: Namespaced
  group: kubevirt.io
  version: v1alpha1
  names:
    kind: OfflineVirtualMachine
    plural: offlinevirtualmachines
    singular: offlinevirtualmachine
    shortNames:
    - ovm
    - ovms
```

Part of the definition of custom resource is a API specification used for
autogenerating the Kubernetes resources: client, lister and watcher.

### Controller

The controller is responsible for watching the change in the registered
offline virtual machines and update the state of the system. It is also
responsible for creating new VirtualMachine when the `running` is set to `true`.
Moreover the controller attaches the `metadata.OwnerReference` to the created
VirtualMachine. With this mechanism it can link the OfflineVirtualMachine to the
VirtualMachine and show combined status.

The controller is designed to be a standalone service running in its own pod.
Since the whole KubeVirt is designed to be modular, this approach allows for
a more flexbility and less codebase in the core. Moreover it can be scaled
up separately if the need arise.

[VirtualMachine]: https://kubevirt.github.io/api-reference/master/definitions.html#_v1_virtualmachine
