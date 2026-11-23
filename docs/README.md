## Technical Overview

Kubernetes allows for extensions to its architecture via
[*custom resources*]( https://Kubernetes.io/docs/concepts/extend-Kubernetes/api-extension/custom-resources/),
which add a new endpoint in the Kubernetes API that stores and retrieves a
collection API objects of a certain kind.
The *custom resources* by themselves only
enable store and retrieve structured data;
to add business logic and specific functionality,
[*custom controllers*]( https://Kubernetes.io/docs/concepts/extend-Kubernetes/) are needed.
*Controllers* are clients of the Kubernetes API-Server that typically read an
object's `.spec`, possibly do things, and then update the object's
`.status`.

KubeVirt uses CRDs, *controllers* and other Kubernetes features, to
represent and manage traditional virtual machines side by side with
containers.

KubeVirt's primary CRD is the VirtualMachine (VM) resource, which manages
the lifecycle of a VirtualMachineInstance (VMI) object that represents a
single virtualized workload that executes once until completion
(i.e., powered off).

### Project Components
The key KubeVirt components are the virt-api, the
virt-controller, the virt-handler, and the virt-launcher.

![KubeVirt components](components.png "Components")

 * **virt-api**: This component provides a HTTP RESTful entrypoint to manage
   the virtual machines within the cluster.
 * **virt-controller**: This component is a Kubernetes controller that
 manages the lifecycle of VMs within the Kubernetes cluster.
 * **virt-handler**: This is a daemon that runs on each Kubernetes node.
It is responsible for monitoring the state of VMIs according to
Kubernetes and ensuring the corresponding libvirt domain is booted or
halted accordingly. To perform these operations, the virt-handler has a
communication channel with each virt-launcher that is used to manage the
lifecycle of the qemu process within the *virt-launcher* pod.
 * **virt-launcher**: There is one per running VMI.
This component directly manages the lifecycle of the qemu process within
the VMI's pod and receives lifecycle commands from virt-handler.

### Scripts

 * `kubevirtci/cluster-up/kubectl.sh`: This is a wrapper around Kubernetes' kubectl command so
   that it can be run directly from this checkout without logging into a node.
 * `hack/virtctl.sh` is a wrapper around `virtctl`. `virtctl` brings all
   virtual machine specific commands with it. It is supplement to `kubectl`.
   e.g. `hack/virtctl.sh console testvm`.
 * `kubevirtci/cluster-up/cli.sh` helps you create ephemeral kubernetes and openshift
   clusters for testing. This is helpful when direct management or access to
   cluster nodes is necessary. e.g. `kubevirtci/cluster-up/cli.sh ssh node01`.

### Makefile Commands

 * `make cluster-up`: This will deploy a fresh environment, the contents of
   `KUBE_PROVIDER` will be used to determine which provider from the `cluster`
   directory will be deployed.
 * `make cluster-sync`: After deploying a fresh environment, or after making
   changes to code in this tree, this command will sync the Pods and DaemonSets
   in the running KubeVirt environment with the state of this tree.
   To run the full sync without pushing images, set `CLUSTER_SYNC_NO_PUSH=true`
   and point `DOCKER_PREFIX` at a node-reachable address for the Bazel registry:
   ```bash
   CLUSTER_SYNC_NO_PUSH=true \
   DOCKER_PREFIX=192.168.1.10:5001/kubevirt \
   BAZEL_REGISTRY_ADVERTISE=192.168.1.10 \
     make cluster-sync KUBEVIRT_PROVIDER=k8s-1.37
   ```
   This builds the image targets used by cluster-sync, including test and
   virt-template images, and serves the Bazel OCI layouts directly. No-push
   cluster-sync currently supports one build architecture per run and does not
   support cross-architecture emulation images. The registry keeps running in
   the Bazel builder container after the sync, so it can serve
   a recreated cluster until stopped with `make cluster-sync-registry-stop`.
   Configure every cluster node to trust the HTTP registry before pulling:

   ```toml
   [[registry]]
   location = "192.168.1.10:5001"
   insecure = true
   ```
 * `make cluster-down`: This will tear down a running KubeVirt environment.
