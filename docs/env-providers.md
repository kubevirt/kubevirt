# Development and Testing Environment Providers

All following providers allow a common workflow:

 * `cluster/up.sh` to create the environment
 * `cluster/down.sh` to stop the environment
 * `cluster/sync.sh` to build the code and deploy it
 * `cluster/deploy.sh` to (re)deploy the code (no provider support needed)
 * `make functests` to run the functional tests against a KubeVirt
 * `cluster/kubectl.sh` to talk to the k8s installation

It is recommended to export the `PROVIDER` vagirable as part of your .bashrc.

## Vagrant

Allows provisioning k8s cluster based on kubeadm. Supports an arbitrary amount
of nodes.

Requires:
 * A working go installation
 * Vagrant installation with libvirt provider
 * Nested virtualization enabled

Usage:

```bash
export PROVIDER=vagrant # choose this provider
export VAGRANT_NUM_NODES=2 # master + two nodes
cluser/up.sh
```

## Local

Allows provisioning a single-master k8s cluster based on latest upstream k8s
code.

Requires:
 * A working go installation
 * A running docker daemon

Usage:

```bash
export PROVIDER=local # choose this provider
cluser/up.sh
```

## New Providers

 * Create a `cluster/$POVIDER` directory
 * Create a `cluster/$PROVIDER/provider.sh` files
 * This file should containe the functions `up`, `build`, `down` and `_kubectl`
