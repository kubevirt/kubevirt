# Components

Make sure to check out the [Glossary](glossary.md) before continuing.

## Overview

KubeVirt consists of a set of services:

                |
        Cluster | (virt-api)  (virt-controller)
                |
    ------------+---------------------------------------
                |
     Kubernetes | (VM TPR)
                |
    ------------+---------------------------------------
                |
      DaemonSet | (libvirtd) (virt-handler) (vm-pod M)
                |

    M: Managed by KubeVirt
    TPR: Third Party Resource

## Example flow: Create and Delete a VM

The following flow illustrates the communication flow between several
(not all) components present in KubeVirt.
In general the communication pattern can be considered to be a
choreography, where all components act by themselves to realize the state
provided by the `VM` objects.

```
Client                     K8s API     VM TPR  Virt Controller         VM Handler
-------------------------- ----------- ------- ----------------------- ----------

                           listen <----------- WATCH /virtualmachines
                           listen <----------------------------------- WATCH /virtualmachines
                                                  |                       |
POST /virtualmachines ---> validate               |                       |
                           create ---> VM ---> observe --------------> observe
                             |          |         v                       v
                           validate <--------- POST /pods              defineVM
                           create       |         |                       |
                             |          |         |                       |
                           schedPod ---------> observe                    |
                             |          |         v                       |
                           validate <--------- PUT /virtualmachines       |
                           update ---> VM ---------------------------> observe
                             |          |         |                    launchVM
                             |          |         |                       |
                             :          :         :                       :
                             |          |         |                       |
DELETE /virtualmachines -> validate     |         |                       |
                           delete ----> * ---------------------------> observe
                             |                    |                    shutdownVM
                             |                    |                       |
                             :                    :                       :
```

**Disclaimer:** The diagram above is not completely accurate, because
there are _temporary workarounds_ in place to avoid bugs and address some
other stuff.

1. A client posts a new VM definition to the K8s API Server.
2. The K8s API Server validates the input and creates a `VM` 3rd party
   resource (TPR) object.
3. The `virt-controller` observes the creation of the new `VM` object
   and creates a corrsponding pod.
4. Kubernetes is scheduling the pod on a host.
5. The `virt-controller` observes that a pod for the `VM` got started and
   updates the `nodeName` field in the`VM` object.
   Now that the `nodeName` is set, the responsibility transitions to the
   `virt-handler` for any further action.
6. The `virt-handler` (_DaemonSet_) observes that a `VM` got assigned to the
   host where it is running on.
6. The `virt-handler` is using the _VM Specification_ and creates a
   corresponding domain using the local `libvirtd` instance.
7. A client deletes the `VM` object through the `virt-api-server`.
8. The `virt-handler` observes the deletion and turns off the domain.

## `virt-api-server`

HTTP API server which serves as the entry point for all virtualization related
flows.

The API Server is taking care to update the virtualization related third party
resources (see below).

As the main entrypoint to KubeVirt it is responsible for defaulting and validation of the provided VM TPRs.

## `VM` (TPR)

VM definitions are kept as third party resources inside the Kubernetes API
server.

The VM definition is defining all properties of the Virtual machine itself,
for example

* Machine type
* CPU type
* Amount of RAM and vCPUs
* Number and type of NICs
* …

## `virt-controller`

From a high-level perspective the virt-controller has all the _cluster wide_
virtualization functionality.

This controller is responsible for monitoring the VM (TPRs) and managing the
associated pods. Currently the controller will make sure to create and manage
the life-cycle of the pods associated to the VM objects.

A VM object will always be associated to a pod during it's life-time, however,
due to i.e. migration of a VM the pod instance might change over time.

## `vm-launcher`

For every VM object one pod is created. This pod's primary container runs the
`vm-launcher`.

Kubernetes or the kubelet is not running the VMs itself. Instead a daemon on
every host in the cluster will take care to launch a VM process for every
pod which is associated to a VM object whenever it is getting scheduled on a host.

The main purpose of the `vm-launcher` is to provide the cgroups and namespaces,
which will be used to host the VM process.
Once a VM process appears in the container, `vm-handler` binds itself to this process and will exit whenever the VM process terminates.
Finally `vm-handler` forwards signals from Kubernetes to the VM process.
With this functionality it is ensured that a pod will never outlive it's VM
process and vice versa.

As of now, a VM process

## `virt-handler`

Every host needs a single instance of `virt-handler`. It can be delivered as a DaemonSet.

Like the `virt-controller`, the `virt-handler` is also reactive and is watching for
changes of the VM object, once detected it will perform all necessary
operations to change a VM to meet the required state.

This behavior is similar to the choreography between the Kubernetes API Server
and the kubelet.

The main areas which `virt-handler` has to cover are:

1. Keep a cluster-level VM spec in sync with a libvirt domain on its host.
2. Report domain state and spec changes to the cluster.
3. Invoke node-centric plugins which can fulfill networking and storage requirements defined in VM specs.

Metrics collection for VMs is not part of `virt-handler`s responsibilities.

## `libvirtd`

On the host an instance of libvirtd is responsible for actually managing the
VM processes.

To integrate the libvirt-managed VM into kubernetes, on startup, the VM is started in the corresponding `vm-launcher` container.

# Additional components

The components above are essential to deliver core virtualization
functionality in your cluster. However fully featured virtual machines require
more than just plain virtualization functionality. Beyond virtualization they
also require reliable storage and networking functionality to be fully usable.

The components below will be providing this additional functionality if the
functionality is not provided by kubernetes itself.

## Storage

We will try to leverage as much of Kubernetes regarding to mounting and preparing images for VM.
However, `virt-handler` may provide a plugin mechanism to allow storage mounting and setup from the host, if the KubeVirt requirements do not fit into the Kubernetes storage scenarios.

Since host side preparation of storage may not be enough, a cluster-wide [Storage Controller](###storage-controller) can be used to prepare storage.

Investigations are still in progress.

###  Storage Controller

Such a controller will not be part of KubeVirt itself.

However KubeVirt might define a Storage TPR along side with a flow description which will allow such a controller seamless integration into KubeVirt.

## Networking

We will try to leverage as much of Kubernetes networking plugin mechanisms (e.g. CNI).
However, `virt-handler` may provide a plugin mechanism to allow network setup on a host, if the KubeVirt requirements do not fit into the Kubernetes storage scenarios.

Since host side preparation of network interfaces may not be enough, a cluster-wide [Network Controller](###network-controller) can be used to prepare the network.

Investigations are still in progress.

## Network Controller

Such a controller will not be part of KubeVirt itself.

However KubeVirt might define a Networking TPR along side with a flow description which will allow such a controller seamless integration into KubeVirt.
