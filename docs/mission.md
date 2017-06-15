# Motivation

There is a wider range of management applications dealing with different
aspects of operating system and workload virtualization, i.e. oVirt (data
center, full OS virt),  OpenStack (cloud, full OS virt) and OpenShift (cloud,
application containers). In terms of infrastructure they all have broadly
similar needs for API resource management, distributed placement and
scheduling, active workload management, etc. Currently they all have
completely separate implementations of these concepts with a high level of
technical duplication.
At the low level, the only area of commonality is sharing of libvirt and KVM
between oVirt and OpenStack.

* This is a poor use of developer resources because multiple projects are
  reinventing the same wheels.
* This is a poor experience for cloud administrators as they have to manually
  partition up their physical machines between the three separate
  applications, and then manage three completely separate pieces of
  infrastructure.
* This is a poor experience for tenant users because they have to learn three
  completely different application APIs and frontends depending on which
  particular type of workload they wish to run

While libvirt has been successful at providing a unified single-host centric
mangement API for virtualization apps to build upon, nothing similar has
arisen to fill the gap for network level compute management to the same degree.


# Aims / goals

The high level goal of this project is to build a Kubernetes native general
purpose system to enable management of KVM, via libvirt, with the Kubernetes
platform. The belief is that over the long term this would enable application
container workloads, data center full OS virt and cloud full OS virt, to be
managed from a single place via the Kubernetes native API.
In other words: Kubernetes would become the single point of entry for cloud no
matter what type of workload is being run.

As an add-on to Kubernetes, it is also a goal to leverage, embrace, and enhance
Kubernetes infrastructure and API objects by default.


# Mission

To design, build, and support an optimized general purpose facility for
managing KVM, via libvirt, as an add-on to the Kubernetes platform.

Which consists of, but is not necessarily limited to:

* Kubernetes resources and services required to spawn and manage virtual
  machines via `kubectl`
* Mechanism to access to graphical console and serial console for virtuall
  machines (websockets proxy)


# FAQ

## Can I perform a 1:1 translation of my libvirt domain xml to a VM Spec?

Probably not. libvirt is intended to be run on a host. And the domain xml is
based on this assumption, this implies that the domain xml allows you to access
host local resources i.e. local paths, host devices, and host device
configurations.
A VM Spec on the other hand is designed to work with cluster resources. And it
does not permit to address host resources.

## Does a VM Spec support all features of libvirt?

No. libvirt has a wide range of features, reaching beyond pure virtualization
fatures, into host, network, and storage management. The API was driven by the
requirements of running virtualization on a host.
A VM Spec however is a VM definition on the _cluster level_, this by itself
means that the specification has different requirements, i.e. it also needs to
include scheduling informations
And KubeVirt specifically builds on Kubernetes, which allows it to reuse the
subsystems for consuming network and storage, which on the other hand means
that the corresponding libvirt features will not be exposed.
Another

## Is KubeVirt a replacement for $MY_VM_MGMT_SYSTEM?

Maybe. The primary goal of KubeVirt is to allow running virtual machines on
top of Kubernetes. It's focused on the virtualization bits.
General virtualization management systems like i.e. OpenStack or oVirt usually
consist of some additional services which take care of i.e. network management,
host provisioning, data warehousing, just to name a few. These services are out
of scope of KubeVirt.
That being said, KubeVirt is intended to be part of a virtualization management
system. It can be seen as an VM cluster runtime, and additional components
provide additional functionality to provide a nice coherent user-experience.

## Is KubeVirt like ClearContainers?
No. ClearContainers are about using VMs to isolate pods or containers on the
container runtime level.
KubeVirt on the other hand is about allowing to manage virtual machines on a
cluster level.
