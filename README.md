# KubeVirt

[![Build Status](https://travis-ci.org/kubevirt/kubevirt.svg?branch=master)](https://travis-ci.org/kubevirt/kubevirt)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubevirt/kubevirt)](https://goreportcard.com/report/github.com/kubevirt/kubevirt)
[![Licensed under Apache License version 2.0](https://img.shields.io/github/license/kubevirt/kubevirt.svg)](https://www.apache.org/licenses/LICENSE-2.0)


**KubeVirt** is a virtual machine management add-on for Kubernetes.
The aim is to provide a common ground for virtualization solutions on top of
Kubernetes.

**Note:** KubeVirt is a heavy work in progress.

# Virtualization extension for Kubernetes

At it's core KubeVirt extends [Kubernetes][k8s] by adding
additional virtualization resource types (especially the `VM` type) through
[Kubernetes's third party resource concept][tpr].
By using this mechanism, the Kubernetes API can be used to manage these `VM`
resources alongside all other resources Kubernetes provides.

The resources themselves are not enough to launch virtual machines.
For this to happen the _functionality and business logic_ needs to be added to
the cluster. The functionality is not added to Kubernetes itself, but rather
added to a Kubernetes cluster by _running_ additional controllers and agents
on an existing cluster.

The necessary controllers and agents are provided by KubeVirt.


## Features

As of today KubeVirt can be used to declarative

 * Create a predefined VM
 * Schedule a VM on a Kubernetes cluster
 * Launch a VM
 * Stop a VM
 * Delete a VM

Example:

[![asciicast](https://asciinema.org/a/96275.png)](https://asciinema.org/a/96275)


## Getting Started

To get started right away please read our
[Getting Started Guide](docs/getting-started.md).


## Documentation

You can learn more about how KubeVirt is designed (and why it is that way),
and learn more about the major components by taking a look at
[our documentation](docs/):

 * [Glossary](docs/glossary.md) - Explaining the most important terms
 * [Architecture](docs/architecture.md) - High-level view on the architetcure
 * [Components](docs/components.md) - Detailed look at all components


## Related resources

 * [Kubernetes][k8s]
 * [Libvirt][libvirt]
 * [Cockpit][cockpit]


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
   [tpr]: http://kubernetes.io/docs/user-guide/thirdpartyresources/
   [ovirt]: https://www.ovirt.org
   [cockpit]: https://cockpit-project.org/
   [libvirt]: https://www.libvirt.org
