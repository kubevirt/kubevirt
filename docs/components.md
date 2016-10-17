# Components

KubeVirt consists of a set of services.

                |
        Cluster | (virt-api-server)  (virt-controller)
                |
    ------------+---------------------------------------
                |
     Kubernetes | (VM TPR)
                |
    ------------+---------------------------------------
                |
      DaemonSet | (libvirtd) (vms-handler) (vm-pod*)
                |

## Virt API Server

HTTP API server which serves as the entry point for all virtualization related
flows.

The API Server is taking care to update the virtualization related third party
resources (see below).
This is effectively mapping an imperative interface (the REST API) onto the
declarative centric resource model.

## VM (TPR)

VM definitions are kept as third party resources inside the Kubernetes API
server.

The VM definition is defining all properties of the Virtual machine itself,
for example

* Machine type
* CPU type
* Amount of RAM and vCPUs
* Number and type of NICs
* …

## Virt Controller

From a high-level perspective the virt-controller has all the _cluster wide_
virtualization functionality.

This controller is responsible for monitoring the VM (TPRs) and managing the
associated pods. Currently the controller will make sure to create and manage
the life-cycle of the pods associated to the VM objects.

A VM object will always be associated to a pod during it's life-time, however,
due to i.e. migration of a VM the pod instance might change over time.

## VM Pods

Every pod associated to a VM object, the pod's primary container is the
`vm-launcher`.

Despite the fact that a VM TPR exists, kubernetes or the kubelet is not running
the VMs itself. Instead a daemon on every host in the cluster will take care to
launch a VM process for every pod which is associated to a VM object whenever
it is getting scheduled on a host.

The main purpose of the vm-launcher is to provide the cgroups and namespaces,
which will be used to host the VM process.
On the other hand this process will terminate whenever and for whatever reason
the VM process goes away.
With this functionality it is ensured that a pod will never outlive it's VM
process and vice versa.

## VMs Handler

From a high-level perspective the `vms-handler` has two big areas to cover:

1. Single VM logic which is taking care of the VM life-cycle on a host
2. Multi VM logic to manage ressources which are shared between multiple
   VMs on a single host (like CPU cores when NUMA is used).

The `vms-handler` is delivered in a DaemonSet and is thus present with a
single instance on every host in a cluster.

Like the virt-controller, the vms-handler is also reactive and watching for
changes of the VM object, once detected it will perform all necessary
operations to change a VM to meet the required state.

This behavior is similar to the choreography between the Kubernetes API Server
and the kubelet.

## libvirtd

On the host an instance of libvirtd is responsible for actually managing the
VM processes.

Due to little additions, the VMs are however running inside some of the
associated pod's namespace and it's cgroups.


# Additional components

The components above are essential to deliver core virtualization
functionality in your cluster. However fully featured virtual machines require
more than just plain virtualization functionality. Beyond virtualization they
also require reliable storage and networking functionality to be fully usable.

The components below will be providing this additional functionality if the
functionality is not provided by kubernetes itself.

## WIP - Storage Controller

WIP - Interface to high-level storage entities/functionality

## WIP - Network Controller

WIP - Interface to high-level storage entities/functionality
