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

### Project Components

 * virt-api: This component provides a HTTP RESTful entrypoint to manage
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
 * `cluster/vm-isolation-check.sh`: This script will run a series of tests to ensure
   the system is set up correctly.
