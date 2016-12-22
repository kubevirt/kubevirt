# KubeVirt

[![Build Status](https://travis-ci.org/kubevirt/kubevirt.svg?branch=master)](https://travis-ci.org/kubevirt/kubevirt)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubevirt/kubevirt)](https://goreportcard.com/report/github.com/kubevirt/kubevirt)

**KubeVirt** is a virtual machine management architecture built around
Kubernetes.
The virtualization capabilities are layered on top of Kubernetes,
existing functionality like scheduling and storage are however directly
consumed from Kubernetes.

KubeVirt aims at beeing able to provide management for fully featured
Virtual Machines, thus VMs where you can tune every single parameter.
These are usually the kind of VMs you find 'classic' datacenter
virtualization environments.

## Getting Started

To get started right away please read out
[Getting Started Guide](docs/getting-started.md).


## Documentation

You can learn more about how KubeVirt is designed (and why it is that way),
and learn more about the major components by taking a look at
[our documentation](docs/):

* [Glossary](docs/glossary.md) - Explaining the most important terms
* [Architecture](docs/architecture.md) - High-level view on the architetcure
* [Components](docs/components.md) - Detailed look at all components


## Technical Overview

Kubernetes allows for extensions to its architecture in the form of 3rd party
resources: <http://kubernetes.io/docs/user-guide/thirdpartyresources/>.
KubeVirt represents virtual machines as 3rd party resources and manages changes
to libvirt domains based on the state of those resources.

This project provides a Vagrant setup with the requisite components already
installed. To boot a vanilla kubernetes environment as base for kubevirt,
simply type `vagrant up` from the root directory of the git tree, which can be
found here:

<https://github.com/kubevirt/kubevirt>

Once the Vagrant provisioning script has completed, run `./cluster/sync.sh` to
build and deploy KubeVirt specific components to the Vagrant nodes.

Note: KubeVirt is built in go. A properly configured go environment is
therefore required. For best results, use this path:
`$GOPATH/src/kubevirt.io/kubevirt/`

### Associated resources

 * Kubernetes
 * Libvirt
 * Cockpit

### Project Components

 * virt-api: This component provides a HTTP RESTfull entrypoint to manage
   the virtual machines within the cluster.
 * virt-controller: This component manages the state of each VM within the
   Kubernetes cluster.
 * virt-handler: This is a daemon that runs on each Kubernetes node. It is
   responsible for monitoring the state of VMs according to Kubernetes and
   ensuring the corresponding libvirt domain is booted or halted accordingly.
 * virt-launcher: This component is a place-holder, one per running VM. Its
   job is to remain running as long as the VM is defined. This simply prevents a
   crash-loop state.
 * ha-proxy: This daemon proxies connections from 192.168.200.2 to the running
   master node--making it possible to establish connections in a consistent
   manner.

### Scripts

 * `cluster/sync.sh`: After deploying a fresh vagrant environment, or after
   making changes to code in this tree, this script will sync the Pods and
   DaemonSets in the running KubeVirt environment with the state of this tree.
 * `cluster/kubectl.sh`: This is a wrapper around Kubernete's kubectl command so
   that it can be run directly from this checkout without logging into a node.
 * `cluster/sync_config.sh`: This script will contact the master node and
   collect its config and kubectl. It is called by sync.sh so does not generally
   need to be run separately.
 * `cluster/quickcheck.sh`: This script will run a series of tests to ensure
   the system is set up correctly.

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
