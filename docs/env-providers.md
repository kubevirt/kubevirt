# Development and Testing Environment Providers

All following providers allow a common workflow:

 * `make cluster-up` to create the environment
 * `make cluster-down` to stop the environment
 * `make cluster-build` to build
 * `make cluster-deploy` to (re)deploy the code (no provider support needed)
 * `make cluster-sync` to build and (re)deploy the code
 * `make functests` to run the functional tests against KubeVirt
 * `cluster/kubectl.sh` to talk to the k8s installation

It is recommended to export the `KUBEVIRT_PROVIDER` variable as part of your `.bashrc`
file.

## Dockerized k8s/ocp clusters

Allows provisioning k8s cluster based on kubeadm. Supports an arbitrary amount
of nodes.

Requires:
 * A working go installation
 * Nested virtualization enabled
 * A running docker daemon

Usage:

```bash
export KUBEVIRT_PROVIDER=k8s-1.10.11 # choose this provider
export KUBEVIRT_NUM_NODES=3 # master + two nodes
make cluster-up
```

## Local

Allows provisioning a single-master k8s cluster based on latest upstream k8s
code.

Requires:
 * A working go installation
 * A running docker daemon

Usage:

```bash
export KUBEVIRT_PROVIDER=local # choose this provider
make cluster-up
```

## External

Uses an existing (external) Kubernetes cluster.

Requires:
 * A working Kubernetes cluster with properly configured worker nodes.
 * A running docker daemon

Usage:

```bash
export KUBEVIRT_PROVIDER=external # choose this provider
make cluster-up
```
## New Providers

 * Create a `cluster/$KUBEVIRT_PROVIDER` directory
 * Create a `cluster/$KUBEVIRT_PROVIDER/provider.sh` files
 * This file has to contain the functions `up`, `build`, `down` and `_kubectl`
 * Have a look at `cluster/k8s-1.10.11/provider.sh` for a reference implementation
