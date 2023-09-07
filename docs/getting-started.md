# Getting Started

A quick start guide to get KubeVirt up and running inside our container-based
development cluster.

**Note**: Docker is used as default container runtime. If you want to use
podman see [PODMAN.md](https://github.com/kubevirt/kubevirtci/blob/main/PODMAN.md).

## I just want it built and run it on my cluster

First, point the `Makefile` to the Docker registry of your choice:

```bash
export DOCKER_PREFIX=index.docker.io/myrepo
export DOCKER_TAG=mybuild
```

Then build the manifests and images:

```bash
make && make push && make manifests
```

Finally, push the manifests to your cluster:

```bash
kubectl create -f _out/manifests/release/kubevirt-operator.yaml
kubectl create -f _out/manifests/release/kubevirt-cr.yaml
```

### Docker Desktop for Mac

The Bazel build system does not support the macOS keychain. Docker uses `osxkeychain`, which is the default [credential helper](https://github.com/docker/docker-credential-helpers) 
for mac.

Modify the `$HOME/.docker/config.json` file to include the following snippet:

```json
{
	"credHelpers": {
		"https://index.docker.io/v1/": ""
	}
}
```

This makes sure that no credential helpers are used for the specified registry and hence the credentials will be stored in the config.json file itself.

Now log in with `docker login`. You will get a warning message saying that no credential helper is configured. Your `$HOME/.docker/config.json` should look like:

```json
{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "XXXXXXXXXX"
		}
	},
	"credsStore": "desktop",
	"credHelpers": {
		"https://index.docker.io/v1/": ""
	}
}
```

## Requirements

### SELinux support

SELinux-enabled nodes need to have [Container-selinux](https://github.com/containers/container-selinux) version 2.170.0 or newer installed.

#### Disabling the custom SELinux policy

By default, a custom SELinux policy gets installed by virt-handler on every node, and it gets used for VMIs that need it.
Currently, the only VMIs using it are the ones that enable passt-based networking.  
However, having KubeVirt install and use a custom SELinux policy is a security concern. It also increases virt-handler boot time 20/30 seconds.  
Therefore, a feature gate was introduced to disable the installation and usage of that custom SELinux policy: `DisableCustomSELinuxPolicy`.  
The side effect is that passt-enabled VMIs will fail to start, but only on nodes that use container-selinux version 2.192.0 or lower.  
container-selinux releases 2.193.0 and newer include the necessary permissions for passt-enabled VMIs to run successfully.

**Note:** adding the `DisableCustomSELinuxPolicy` feature gate to an existing cluster will disable the use of the custom policy for new VMIs,
but will **not** automatically uninstall the policy from the nodes. That can be done manually if needed, by running `semodule -r virt_launcher` on every node.

## Building

The KubeVirt build system runs completely inside Docker. 
In order to build KubeVirt you need to have `docker` and `rsync` installed. 
You also need to have `docker` running, and have the 
[permissions](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user) 
to access it.

**Note:** For running KubeVirt in the dockerized cluster, **nested
virtualization** must be enabled - [see here for instructions for Fedora](https://docs.fedoraproject.org/en-US/quick-docs/using-nested-virtualization-in-kvm/index.html).
As an alternative [software emulation](software-emulation.md) can be allowed.
Enabling nested virtualization should be preferred.

### Dockerized environment

Runs master and nodes containers. Each one of them runs virtual machines via QEMU.
In addition it runs `dnsmasq` and Docker registry containers.

### Compatibility

The minimum compatible Kubernetes version is 1.15.0. Important features required
for scheduling and memory are missing or incompatible with previous versions.

### Compile and run it

To build all required artifacts and launch the
dockerized environment, clone the KubeVirt repository, `cd` into it, and:

```bash
# Build and deploy KubeVirt on Kubernetes in our vms inside containers
# export KUBEVIRT_PROVIDER=k8s-1.20 #  uncomment to use a non-default KUBEVIRT_PROVIDER
make cluster-up
make cluster-sync
```

This will create a virtual machine called `node01` which acts as node and control-plane. To create
more nodes which will register themselves on control-plane, you can use the
`KUBEVIRT_NUM_NODES` environment variable. This would create a control-plane and one
node:

```bash
export KUBEVIRT_NUM_NODES=2 # schedulable control-plane + one additional node
make cluster-up
```

You can use the `KUBEVIRT_MEMORY_SIZE` environment 
variable to increase memory size per node. Normally you do not need it, 
because default node memory size is set.

```bash
export KUBEVIRT_MEMORY_SIZE=8192M # node has 8GB memory size
make cluster-up
```

You can use the `FEATURE_GATES` environment variable to enable one or more feature gates provided by KubeVirt. The 
list of feature gates (which evolve in time) can be checked directly from the 
[source code](https://github.com/kubevirt/kubevirt/blob/main/pkg/virt-config/feature-gates.go).

```bash
# export FEATURE_GATES=<feature-gate-1>,<feature-gate-2>
# e.g. to enable Sidecar and HotplugNICs feature gates run below
$ export FEATURE_GATES=Sidecar,HotplugNICs
$ make cluster-sync
```

**Note:** If you see the error below, 
check if the MTU of the container and the host are the same. 
If not, try to adjust them to be the same. 
See [issue 2667](https://github.com/kubevirt/kubevirt/issues/2667)
for more detailed info.
```
# ./cluster-up/kubectl.sh get pods --all-namespaces
NAMESPACE     NAME                                      READY   STATUS             RESTARTS   AGE
cdi           cdi-operator-5db567b486-grtk9             0/1     ImagePullBackOff   0          42m

Back-off pulling image "kubevirt/cdi-operator:v1.10.1"
```

To destroy the created cluster, type:

```
make cluster-down
```

**Note:** Whenever you type `make cluster-down && make cluster-up`, you will
have a completely fresh cluster to play with.

#### Sync specific components

**Note:** The following is meant for allowing faster iteration on small changes to components that support it.
Not every component is that simply exchangeable without a full re-deploy. Always test with the final SHA based method in the end.

In situations where you just want to work on a single component and rollout updates
without re-deploying the whole environment, you can tell kubevirt to deploy using tags.

```sh
export KUBEVIRT_ONLY_USE_TAGS=true
```

After this any `make cluster-sync` will use the `DOCKER_TAG` for pulling images instead of SHAs.
This means you can simply rebuild the component that changed and then kill the respective pods to
cause a fresh pull:

```sh
PUSH_TARGETS='virt-api' ./hack/bazel-push-images.sh
kubectl delete po -n kubevirt -l kubevirt.io=virt-api
```

Once the respective component is back, it should be using your new build.

### Accessing the containerized nodes via ssh

Based on the used cluster, node names might be different.
You can get the names from following command:

```bash
# cluster-up/kubectl.sh get nodes
NAME     STATUS   ROLES                   AGE   VERSION
node01   Ready    control-plane,worker    13s   v1.18.3
```

Then you can execute the following command to access the node:
```
# ./cluster-up/ssh.sh node01
[vagrant@node01 ~]$
```

### Automatic Code Generation

Some of the code in our source tree is auto-generated (see `git ls-files|grep '^pkg/.*generated.*go$'`).
On certain occasions (but not when building git-cloned code), you need to regenerate it
with:

```bash
make generate
```

Typical cases where code regeneration should be triggered are:

 * When changing APIs, REST paths or their comments (gets reflected in the API documentation, clients, generated cloners...)
 * When changing mocked interfaces (the mock generator needs to update the generated mocks)

We have a check in our CI system so that you do not miss when `make generate` needs to be called.

 * Another case is when kubevirtci is updated, in order to vendor cluster-up run `hack/bump-kubevirtci.sh` and then
   `make generate`

### Testing

After a successful build you can run the *unit tests*:

```bash
    make
    make test
```

They do not need a running KubeVirt environment to succeed.
To run the *functional tests*, make sure you have set
up a dockerized environment. Then run

```bash
    make cluster-sync # synchronize with your code, if necessary
    make functest # run the functional tests against the dockerized VMs
```

If you would like to run specific functional tests only, you can leverage `ginkgo`
command line options as follows (run a specified suite):

```
    FUNC_TEST_ARGS='-focus-file=vmi_networking_test' make functest
```

In addition, if you want to run a specific test or tests you can prepend any `Describe`,
`Context` and `It` statements of your test with an `F` and Ginkgo will only run items
that are marked with the prefix. Remember to remove the prefix before issuing
your pull request.

For additional information check out the [Ginkgo focused specs documentation](https://onsi.github.io/ginkgo/#focused-specs)

## Use

Congratulations, you are still with us and you have built KubeVirt.

Now it is time to get hands on and give it a try.

### Create a first Virtual Machine

Finally start a VMI called `vmi-ephemeral`:

```bash
    # This can be done from your GIT repo, no need to log into a VMI

    # Create a VMI
    ./cluster-up/kubectl.sh create -f examples/vmi-ephemeral.yaml

    # Sure? Let's list all created VMIs
    ./cluster-up/kubectl.sh get vmis

    # Enough, let's get rid of it
    ./cluster-up/kubectl.sh delete -f examples/vmi-ephemeral.yaml


    # You can actually use kubelet.sh to introspect the cluster in general
    ./cluster-up/kubectl.sh get pods

    # To check the running kubevirt services you need to introspect the `kubevirt` namespace:
    ./cluster-up/kubectl.sh -n kubevirt get pods
```

This will start a VMI on control-plane or one of the running nodes with a macvtap and a
tap networking device attached.

#### Example

```bash
$ ./cluster-up/kubectl.sh create -f examples/vmi-ephemeral.yaml
vm "vmi-ephemeral" created

$ ./cluster-up/kubectl.sh get pods
NAME                              READY     STATUS    RESTARTS   AGE
virt-launcher-vmi-ephemeral9q7es  1/1       Running   0          10s

$ ./cluster-up/kubectl.sh get vmis
NAME            AGE   PHASE     IP              NODENAME
vmi-ephemeral   11s   Running   10.244.140.77   node02

$ ./cluster-up/kubectl.sh get vmis -o json
{
    "kind": "List",
    "apiVersion": "v1",
    "metadata": {},
    "items": [
        {
            "apiVersion": "kubevirt.io/v1alpha2",
            "kind": "VirtualMachine",
            "metadata": {
                "creationTimestamp": "2016-12-09T17:54:52Z",
                "labels": {
                    "kubevirt.io/nodeName": "master"
                },
                "name": "vmi-ephemeral",
                "namespace": "default",
                "resourceVersion": "102534",
                "selfLink": "/apis/kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances/testvm",
                "uid": "7e89280a-be62-11e6-a69f-525400efd09f"
            },
            "spec": {
    ...
```

### Accessing the Domain via VNC

First make sure you have `remote-viewer` installed. On Fedora run:

```bash
dnf install virt-viewer
```

Windows users can [download remote-viewer from virt-manager.org](https://virt-manager.org/download.html), and may need
to add virt-viewer installation folder to their `PATH`.

Then, after you made sure that the VMI `vmi-ephemeral` is running, type:

```
cluster-up/virtctl.sh vnc vmi-ephemeral
```

This will start a remote session with `remote-viewer`.

`cluster-up/virtctl.sh` is a wrapper around `virtctl`. `virtctl` brings all
virtual machine specific commands with it and is a supplement to `kubectl`.

**Note:** If accessing your cluster through ssh, be sure to forward your X11 session in order to launch `virtctl vnc`.

### Bazel and KubeVirt

#### Build.bazel merge conflicts

You may encounter merge conflicts in `BUILD.bazel` files when creating pull
requests. Normally you can resolve these conflicts extremely easy by simply
accepting the new upstream version of the files and run `make` again. That will
update the build files with your changes.

#### Build.bazel build failures when switching branches

In case you work on two or more branches, `make generate` for example might fail,
the reason is there is a Bazel server in the background, and when the base image changes,
it should be auto restarted, the detection does not always work perfectly.
To solve it, run `docker stop kubevirt-bazel-server`, which will stop the Bazel server.
