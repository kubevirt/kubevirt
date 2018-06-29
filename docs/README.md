## Technical Overview

Kubernetes allows for extensions to its architecture in the form of custom
resources: <https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/>.
KubeVirt represents virtual machines as custom resources and manages changes
to libvirt domains based on the state of those resources.

### Project Components

 * virt-api: This component provides a HTTP RESTful entrypoint to manage
   the virtual machines within the cluster.
 * virt-controller: This component manages the state of each VMI within the
   Kubernetes cluster.
 * virt-handler: This is a daemon that runs on each Kubernetes node. It is
   responsible for monitoring the state of VMIs according to Kubernetes and
   ensuring the corresponding libvirt domain is booted or halted accordingly.
 * virt-launcher: This component is a place-holder, one per running VMI. It
   contains the running VMI and remains running as long as the VMI is defined.

### Scripts

 * `cluster/kubectl.sh`: This is a wrapper around Kubernetes' kubectl command so
   that it can be run directly from this checkout without logging into a node.
 * `cluster/virtctl.sh` is a wrapper around `virtctl`. `virtctl` brings all
   virtual machine specific commands with it. It is supplement to `kubectl`.
   e.g. `cluster/virtctl.sh console testvm`.
 * `cluster/cli.sh` helps you creating ephemeral kubernetes and openshift
   clusters for testing. This is helpful when direct management or access to
   cluster nodes is necessary. e.g. `cluster/cli.sh ssh node01`.

### Makefile Commands

 * `make cluster-up`: This will deploy a fresh environment, the contents of
   `KUBE_PROVIDER` will be used to determine which provider from the `cluster`
   directory will be deployed.
 * `make cluster-sync`: After deploying a fresh environment, or after making
   changes to code in this tree, this command will sync the Pods and DaemonSets
   in the running KubeVirt environment with the state of this tree.
 * `make cluster-down`: This will tear down a running KubeVirt enviornment.
