# KubeVirt

[![Build Status](https://travis-ci.org/kubevirt/kubevirt.svg?branch=master)](https://travis-ci.org/kubevirt/kubevirt)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubevirt/kubevirt)](https://goreportcard.com/report/github.com/kubevirt/kubevirt)
[![Licensed under Apache License version 2.0](https://img.shields.io/github/license/kubevirt/kubevirt.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Coverage Status](https://img.shields.io/coveralls/kubevirt/kubevirt/master.svg)](https://coveralls.io/github/kubevirt/kubevirt?branch=master)
[![Visit our IRC channel](https://kiwiirc.com/buttons/irc.freenode.net/kubevirt.png)](https://kiwiirc.com/client/irc.freenode.net/#kubevirt)

**KubeVirt** is a virtual machine management add-on for Kubernetes.
The aim is to provide a common ground for virtualization solutions on top of
Kubernetes.

**Note:** KubeVirt is a heavy work in progress.

# Introduction

## Virtualization extension for Kubernetes

At its core, KubeVirt extends [Kubernetes][k8s] by adding
additional virtualization resource types (especially the `VM` type) through
[Kubernetes's Custom Resource Definitions API][crd].
By using this mechanism, the Kubernetes API can be used to manage these `VM`
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
 * Stop a VM
 * Delete a VM

Example:

[![asciicast](https://asciinema.org/a/96275.png)](https://asciinema.org/a/96275)


# To start using KubeVirt

Try our quickstart at [kubevirt.io](http://kubevirt.io/get_kubevirt/).

See our user documentation at [kubevirt.io/docs](http://kubevirt.io/user-guide).

# To start developing KubeVirt

To set up a development environment please read our
[Getting Started Guide](docs/getting-started.md). To learn how to contribute, please read our [contribution guide](https://github.com/kubevirt/kubevirt/blob/master/CONTRIBUTING.md).

You can learn more about how KubeVirt is designed (and why it is that way),
and learn more about the major components by taking a look at
[our developer documentation](docs/):

 * [Architecture](docs/architecture.md) - High-level view on the architecture
 * [Components](docs/components.md) - Detailed look at all components
 * [API Reference](https://www.kubevirt.io/api-reference/)


# Community

If you got enough of code and want to speak to people, then you got a couple
of options:

* Follow us on [Twitter](https://twitter.com/kubevirt)
* Chat with us on IRC via [#kubevirt @ irc.freenode.net](https://kiwiirc.com/client/irc.freenode.net/kubevirt)
* Discuss with us on the [kubevirt-dev Google Group](https://groups.google.com/forum/#!forum/kubevirt-dev)
* Stay informed about designs and upcoming events by watching our [community content](https://github.com/kubevirt/community/)
* Take a glance at [future planning](https://trello.com/b/50CuosoD/kubevirt)

## Related resources

 * [Kubernetes][k8s]
 * [Libvirt][libvirt]
 * [Cockpit][cockpit]
 * [Kubevirt-ansible][kubevirt-ansible]

## Submitting patches

When sending patches to the project, the submitter is required to certify that
they have the legal right to submit the code. This is achieved by adding a line

    Signed-off-by: Real Name <email@address.com>

to the bottom of every commit message. Existence of such a line certifies
that the submitter has complied with the Developer's Certificate of Origin 1.1,
(as defined in the file docs/developer-certificate-of-origin).

This line can be automatically added to a commit in the correct format, by
using the '-s' option to 'git commit'.

## License

KubeVirt is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.txt).

    Copyright 2016

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.

[//]: # (Reference links)
   [k8s]: https://kubernetes.io
   [crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/
   [ovirt]: https://www.ovirt.org
   [cockpit]: https://cockpit-project.org/
   [libvirt]: https://www.libvirt.org
   [kubevirt-ansible]: https://github.com/kubevirt/kubevirt-ansible
