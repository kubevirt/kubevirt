Motivation
==========

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

Aims / goals
============

The high level goal of this project is to build a Kubernetes native general
purpose system to enable management of KVM, via libvirt, with the Kubernetes
platform. The belief is that over the long term this would enable application
container workloads, data center full OS virt and cloud full OS virt, to be
managed from a single place via the Kubernetes native API. IOW Kubernetes
would become the single point of entry for cloud no matter what type of
workload is being run.

The VM management system could also serve as a building block to automate the
provisioning of new virtual machines for running isolated container workloads.
ie, some kubernetes compute nodes would in fact be running inside VMs running
on other kubernetes compute nodes.

Mission
=======

To design, build, and support an optimized general purpose facility for
managing KVM, via libvirt, as an add-on to the Kubernetes platform.

Deliverables
============

* Kubernetes resources and services required to spawn and manage virtual
  machines via `kubectl`
* Access to graphical console and serial console for virtuall machines
  (websockets proxy)
* Graphical interface to view managed virtual machines & resources (cockpit)
