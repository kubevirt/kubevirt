**HyperConverged Cluster Operator** is an Operator pattern for managing multi-operator products.
Specifcally, the HyperConverged Cluster Operator manages the deployment of KubeVirt,
Containerized Data Importer (CDI), Virtual Machine import operator and Cluster Network Addons (CNA) operators.

**KubeVirt** is a virtual machine management add-on for Kubernetes.
The aim is to provide a common ground for virtualization solutions on top of
Kubernetes.

# Virtualization extension for Kubernetes

At its core, KubeVirt extends [Kubernetes](https://kubernetes.io) by adding
additional virtualization resource types (especially the `VirtualMachine` type) through
[Kubernetes's Custom Resource Definitions API](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/).
By using this mechanism, the Kubernetes API can be used to manage these `VirtualMachine`
resources alongside all other resources Kubernetes provides.

The resources themselves are not enough to launch virtual machines.
For this to happen the _functionality and business logic_ needs to be added to
the cluster. The functionality is not added to Kubernetes itself, but rather
added to a Kubernetes cluster by _running_ additional controllers and agents
on an existing cluster.

The necessary controllers and agents are provided by KubeVirt.

As of today KubeVirt can be used to declaratively

  * Create a predefined VM
  * Schedule a VM on a Kubernetes cluster
  * Launch a VM
  * Migrate a VM
  * Stop a VM
  * Delete a VM

# Start using KubeVirt

  * Try our quickstart at [kubevirt.io](http://kubevirt.io/get_kubevirt/).
  * See our user documentation at [kubevirt.io/docs](http://kubevirt.io/user-guide).

# Start developing KubeVirt

To set up a development environment please read our
[Getting Started Guide](https://github.com/kubevirt/kubevirt/blob/master/docs/getting-started.md).
To learn how to contribute, please read our [contribution guide](https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md).

You can learn more about how KubeVirt is designed (and why it is that way),
and learn more about the major components by taking a look at our developer documentation:

  * [Architecture](https://github.com/kubevirt/kubevirt/blob/master/docs/architecture.md) - High-level view on the architecture
  * [Components](https://github.com/kubevirt/kubevirt/blob/master/docs/components.md) - Detailed look at all components
  * [API Reference](https://github.com/kubevirt/kubevirt/blob/master/https://www.kubevirt.io/api-reference/)

# Community

If you got enough of code and want to speak to people, then you got a couple of options:

  * Follow us on [Twitter](https://twitter.com/kubevirt)
  * Chat with us in the #virtualization channel of the [Kubernetes Slack](https://slack.k8s.io/)
  * Discuss with us on the [kubevirt-dev Google Group](https://groups.google.com/forum/#!forum/kubevirt-dev)
  * Stay informed about designs and upcoming events by watching our [community content](https://github.com/kubevirt/community/)

# License

KubeVirt is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.txt).
