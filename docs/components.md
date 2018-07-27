# Components

## Overview

KubeVirt consists of a set of services:

                |
        Cluster | (virt-controller)
                |
    ------------+---------------------------------------
                |
     Kubernetes | (VMI CRD)
                |
    ------------+---------------------------------------
                |
      DaemonSet | (virt-handler) (vm-pod M)
                |

    M: Managed by KubeVirt
    CRD: Custom Resource Definition

## Example flow: Create and Delete a VMI

The following flow illustrates the communication flow between several
(not all) components present in KubeVirt.
In general the communication pattern can be considered to be a
choreography, where all components act by themselves to realize the state
provided by the `VMI` objects.

```
Client                     K8s API     VMI CRD  Virt Controller         VMI Handler
-------------------------- ----------- ------- ----------------------- ----------

                           listen <----------- WATCH /virtualmachines
                           listen <----------------------------------- WATCH /virtualmachines
                                                  |                       |
POST /virtualmachines ---> validate               |                       |
                           create ---> VMI ---> observe --------------> observe
                             |          |         v                       v
                           validate <--------- POST /pods              defineVMI
                           create       |         |                       |
                             |          |         |                       |
                           schedPod ---------> observe                    |
                             |          |         v                       |
                           validate <--------- PUT /virtualmachines       |
                           update ---> VMI ---------------------------> observe
                             |          |         |                    launchVMI
                             |          |         |                       |
                             :          :         :                       :
                             |          |         |                       |
DELETE /virtualmachines -> validate     |         |                       |
                           delete ----> * ---------------------------> observe
                             |                    |                    shutdownVMI
                             |                    |                       |
                             :                    :                       :
```

**Disclaimer:** The diagram above is not completely accurate, because
there are _temporary workarounds_ in place to avoid bugs and address some
other stuff.

1. A client posts a new VMI definition to the K8s API Server.
2. The K8s API Server validates the input and creates a `VMI` custom resource
   definition (CRD) object.
3. The `virt-controller` observes the creation of the new `VMI` object
   and creates a corrsponding pod.
4. Kubernetes is scheduling the pod on a host.
5. The `virt-controller` observes that a pod for the `VMI` got started and
   updates the `nodeName` field in the`VMI` object.
   Now that the `nodeName` is set, the responsibility transitions to the
   `virt-handler` for any further action.
6. The `virt-handler` (_DaemonSet_) observes that a `VMI` got assigned to the
   host where it is running on.
6. The `virt-handler` is using the _VMI Specification_ and signals the creation
   of the corresponding domain using a `libvirtd` instance in the VMI's pod.
7. A client deletes the `VMI` object through the `virt-api-server`.
8. The `virt-handler` observes the deletion and turns off the domain.

## `virt-api-server`

HTTP API server which serves as the entry point for all virtualization related
flows.

The API Server is taking care to update the virtualization related custom
resource definition (see below).

As the main entrypoint to KubeVirt it is responsible for defaulting and validation of the provided VMI CRDs.

## `VMI` (CRD)

VMI definitions are kept as custom resource definitions inside the Kubernetes API
server.

The VMI definition is defining all properties of the Virtual machine itself,
for example

* Machine type
* CPU type
* Amount of RAM and vCPUs
* Number and type of NICs
* …

## `virt-controller`

From a high-level perspective the virt-controller has all the _cluster wide_
virtualization functionality.

This controller is responsible for monitoring the VMI (CRDs) and managing the
associated pods. Currently the controller will make sure to create and manage
the life-cycle of the pods associated to the VMI objects.

A VMI object will always be associated to a pod during it's life-time, however,
due to i.e. migration of a VMI the pod instance might change over time.

## `virt-launcher`

For every VMI object one pod is created. This pod's primary container runs the
`virt-launcher` KubeVirt component.

Kubernetes or the kubelet is not running the VMIs itself. Instead a daemon on
every host in the cluster will take care to launch a VMI process for every
pod which is associated to a VMI object whenever it is getting scheduled on a
host.

The main purpose of the `virt-launcher` Pod is to provide the cgroups and
namespaces, which will be used to host the VMI process.

`virt-handler` signals `virt-launcher` to start a VMI by passing the VMI's CRD object
to `virt-launcher`. `virt-launcher` then uses a local libvirtd instance within its
container to start the VMI. From there `virt-launcher` monitors the VMI process and
terminates once the VMI has exited.

If the Kubernetes runtime attempts to shutdown the `virt-launcher` pod before the
VMI has exited, `virt-launcher` forwards signals from Kubernetes to the VMI
process and attempts to hold off the termination of the pod until the VMI has
shutdown successfully.

## `virt-handler`

Every host needs a single instance of `virt-handler`. It can be delivered as a DaemonSet.

Like the `virt-controller`, the `virt-handler` is also reactive and is watching for
changes of the VMI object, once detected it will perform all necessary
operations to change a VMI to meet the required state.

This behavior is similar to the choreography between the Kubernetes API Server
and the kubelet.

The main areas which `virt-handler` has to cover are:

1. Keep a cluster-level VMI spec in sync with a corresponding libvirt domain.
2. Report domain state and spec changes to the cluster.
3. Invoke node-centric plugins which can fulfill networking and storage requirements defined in VMI specs.


## `libvirtd`

An instance of `libvirtd` is present in every VMI pod. `virt-launcher` uses libvirtd
to manage the life-cycle of the VMI process.

# Additional components

The components above are essential to deliver core virtualization
functionality in your cluster. However fully featured virtual machines require
more than just plain virtualization functionality. Beyond virtualization they
also require reliable storage and networking functionality to be fully usable.

The components below will be providing this additional functionality if the
functionality is not provided by kubernetes itself.

## Storage

We will try to leverage as much of Kubernetes regarding to mounting and preparing images for VMI.
However, `virt-handler` may provide a plugin mechanism to allow storage mounting and setup from the host, if the KubeVirt requirements do not fit into the Kubernetes storage scenarios.

Since host side preparation of storage may not be enough, a cluster-wide [Storage Controller](###storage-controller) can be used to prepare storage.

Investigations are still in progress.

###  Storage Controller

Such a controller will not be part of KubeVirt itself.

However KubeVirt might define a Storage CRD along side with a flow description which will allow such a controller seamless integration into KubeVirt.

## Networking

We will try to leverage as much of Kubernetes networking plugin mechanisms (e.g. CNI).
However, `virt-handler` may provide a plugin mechanism to allow network setup on a host, if the KubeVirt requirements do not fit into the Kubernetes storage scenarios.

Since host side preparation of network interfaces may not be enough, a cluster-wide [Network Controller](###network-controller) can be used to prepare the network.

Investigations are still in progress.

## Network Controller

Such a controller will not be part of KubeVirt itself.

However KubeVirt might define a Networking CRD along side with a flow description which will allow such a controller seamless integration into KubeVirt.
