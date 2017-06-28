# Overview

KubeVirt is a tool for scheduling and managing the lifecycle of Virtual 
Machines (VMs) across a distributed set of nodes. In a broad sense KubeVirt
is targeting all use-cases involving VM placement and management in the
datacenter.

This means KubeVirt is being built with a feature set capable of delivering
both the scale out functionality that powers cloud IaaS projects as well as the
high availability features desired for on premise datacenter virtualization.
Since KubeVirt is built upon Kubernetes, this also introduces the option of a
hybrid cluster where both container and virtualized workloads coexist.

This may sound ambitious and slightly confusing considering how many other
projects exist in the VM scheduling space. By comparing KubeVirt to these other
projects, it should be clear what KubeVirt’s scope is and where it fits in the
ecosystem.

# KubeVirt vs. Other Projects

## Kubernetes

Kubernetes is an open source project built to automate deployment and lifecycle
management of containerized applications.

KubeVirt is a drop in addon that can be applied to any to Kubernetes cluster
that gives Kubernetes the ability to manage VMs.

## OpenStack

OpenStack is an open source IaaS platform built of various components to provide
a wide range of features such as compute (VM scheduling), networking, block
storage, object storage, ect.

KubeVirt is a single component focused entirely on VM scheduling. Network and
block storage are something that KubeVirt consumes from other systems and not
something KubeVirt attempts to define. Similar to the way Libvirt has become a
standard for managing VMs on a local node across multiple projects, we hope to
see KubeVirt become the widely adopted standard for scheduling VMs. It is
theoretically possible OpenStack could adopt KubeVirt for VM scheduling.

## Nova

Nova is the open source component of OpenStack responsible for VM scheduling.
Nova provides an abstraction layer that allows multiple virtualization
technologies to be utilized. This means Nova is maintaining drivers for
technologies such as Xen, VMware, KVM, Hyper-V, and LXC.

KubeVirt has made the choice to focus on KVM managed by Libvirt. Initially this
may sound like it is limiting KubeVirt’s functionality, but in practice KVM and
Libvirt have emerged as the leading standard in open source virtualization. By
focusing on this technology stack we aren’t limited by creating a one size fits
all virtualization technology abstraction. We can fully support a feature set
that leverages all that Libvirt and KVM have to offer.

## oVirt

oVirt is an open source virtualization management platform that provides VM
scheduling, network and storage. One of the fundamental differences between
oVirt and other similar projects like OpenStack is the requirement for
infrastructure High Availability.

Cloud infrastructure provided by OpenStack or public clouds like EC2 and GCE
make the assumption that applications running on their platforms have resiliency
built in. oVirt is for the use cases where applications require strict HA
guarantees at the infrastructure level. With oVirt, data integrity is enforced
by node level fencing.

KubeVirt aims to provide a feature set capable of providing the strong
consistency guarantees required for oVirt VMs as well as the scale out
functionality desired for cloud IaaS VMs. Compared to oVirt, KubeVirt could
replace the scheduling component of oVirt and consume the network and storage
resources oVirt already provides.

## Libvirt

Libvirt is an open source project that provides a feature set for handling
VM lifecycle actions (start, stop, pause, save, restore, and live migration)
as well as VM network and storage interface management on a local node. Libvirt
features are provided by the libvirtd daemon which exposes an API client
applications can invoke.

KubeVirt is leveraging the libvirt daemon for management of KVM VMs. Although
KubeVirt could invoke KVM directly, this would require KubeVirt reimplementing
functionality that libvirt already provides.

## AWS EC2 and Google GCE

EC2 and GCE are closed source public cloud IaaS offerings provided by AWS and
Google. This means users are locked into using datacenters and pricing models
provided by these companies. Both of these projects provide an API allowing
users to schedule and manage VMs within their datacenters. Those APIs are
tightly coupled with other AWS and Google services that provide networking,
storage, and other capabilities.

It should be obvious that KubeVirt is not a service. Kubevirt is an opensource
project that focuses solely on scheduling VMs onto a cluster of machines and is
not tightly coupled to any network or storage backend. When compared to EC2 and
GCE, we view KubeVirt’s scope to be that of the scheduling component provided
by these services.

We aim to provide the kind of feature set and flexibility that would
allow IaaS providers to replace their underlying VM scheduling component with
KubeVirt.

# Use Cases

Hopefully by understanding KubeVirt’s relationship to other projects the scope
of KubeVirt is more obvious. KubeVirt is not trying to compete with any of
these projects, but instead it is a tool these projects can adopt for scheduling
VMs rather than maintaining their own custom logic.

To distill all of this down to more precise terms, these are the use cases we
are initially focusing on.

**Cloud Virtualization** - A feature set capable of managing VM scale out with
the type of abstractions people have come to expect from cloud IaaS APIs
provided by OpenStack, EC2, and GCE.

**Datacenter Virtualization** - Providing the strong infrastructure consistency
guarantees required for projects managing pet VMs.

**Kubernetes Trusted Workloads** - The ability to run virtualized workloads in a
Kubernetes cluster that are unsafe to run without the security guarantees a
hypervisor provides.

**Combining Container and Virtualized Workloads** - The ability to schedule both
container and virtualized workloads on the same Kubernetes cluster.
