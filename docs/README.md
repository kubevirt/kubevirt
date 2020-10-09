## Technical Overview

Kubernetes allows for extensions to its architecture via
[*custom resources*]( https://Kubernetes.io/docs/concepts/extend-Kubernetes/api-extension/custom-resources/),
which add a new endpoint in the Kubernetes API that stores and retrieves a
collection API objects of a certain kind. However, the *custom resources*
by themselves only enable store and retrieve structured data. To add
business logic and specific functionality into Kubernetes, it is necessary
to use
[*custom controllers*]( https://Kubernetes.io/docs/concepts/extend-Kubernetes/),
which are clients of the Kubernetes API-Server that typically read an
object's `.spec`, possibly do things, and then update the object's
`.status`.

KubeVirt uses CRDs, *controllers* and other Kubernetes features, to
represent and manage traditional virtual machines side by side with
containers.

KubeVirt's primary CRD is the VirtualMachine (VM) resource, which contains
a collection of VirtualMachineInstance (VMI) objects, which shares
similarity with the Pod concept. A VMI represents a single virtualized
workload that executes once until completion (i.e., powered off). In
addition to the VMI, the key KubeVirt components are the virt-api, the
virt-controller, the virt-handler, and the virt-launcher.

### Project Components

 * **virt-api**: This component provides a HTTP RESTful entrypoint to manage
   the virtual machines within the cluster.
 * **virt-controller**: This component is a Kubernetes Operator that
 manages the state of each VMI within the Kubernetes cluster. When new VM
 objects are submitted to the Kubernetes API-Server, this controller takes
 notice and creates the pod in which the VM will run and delegates the
 other management operations to the *virt-handler* component.
 * **virt-handler**: This is a daemon that runs on each Kubernetes node. It is
   responsible for monitoring the state of VMIs according to Kubernetes and
   ensuring the corresponding libvirt domain is booted or halted accordingly.
 * **virt-launcher**: This component is a place-holder, one per running VMI. It
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
 * `make cluster-down`: This will tear down a running KubeVirt environment.
