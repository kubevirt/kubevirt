# Development and Testing Environment Providers

All following providers allow a common workflow:

 * `make cluster-up` to create the environment
 * `make cluster-down` to stop the environment
 * `make cluster-build` to build
 * `make cluster-deploy` to (re)deploy the code (no provider support needed)
 * `make cluster-sync` to build and (re)deploy the code
 * `make cluster-sync-fast` to build and roll only changed core images on an
   installed KubeVirt cluster. Run `make cluster-sync` once to bootstrap a
   fresh cluster. It publishes changed images to the provider's configured
   registry (the local registry for kind). Configure components, workloads,
   rollout flows, and combination rules in `hack/cluster-sync-fast.yaml`; select
   a subset with, for example, `KUBEVIRT_FAST_COMPONENTS="virt-api virt-handler"
   make cluster-sync-fast` (or use the generic `CLUSTER_SYNC_FAST_COMPONENTS`).
   This host-side command requires Mike Farah `yq` v4
   to read the YAML manifest. Core images follow `DOCKER_TAG`; virt-template
   images follow `VIRT_TEMPLATE_VERSION`, matching `cluster-sync`. The defaults
   cover installed control-plane workloads plus the launcher/export images
   selected by virt-controller. Set `CLUSTER_SYNC_FAST_DRY_RUN=true` with
   `CLUSTER_SYNC_FAST_COMPONENTS` to preview rollout order without a cluster,
   build, or push; selected components are treated as changed. The default integration
   uses the KubeVirt Bazel and kubevirtci helpers. Set `hooks.file` in the YAML
   manifest to use another adapter. `CLUSTER_SYNC_FAST_HOOKS` can point to a Bash
   overlay that overrides individual functions. The adapter provides
   `cluster_sync_fast_cluster_ready`, `cluster_sync_fast_build`, and
   `cluster_sync_fast_push`; the readiness hook may set `namespace` and
   `kubectl_cmd`, the build hook receives component names, build targets, and
   digest paths and prints `FAST_SYNC_DIGEST=component:digest` lines, and the
   push hook receives image names, tag, and alternate tag. Custom flow actions
   can be handled by an optional `cluster_sync_fast_action` hook with arguments:
   action, component, workload, namespace, kubectl command, and timeout.
 * `make cluster-sync-fast-test` to run the Bats unit tests for this runner. It
   requires Bats and Mike Farah `yq` v4.
 * `make functests` to run the functional tests against KubeVirt
 * `kubevirtci/cluster-up/kubectl.sh` to talk to the k8s installation

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
export KUBEVIRT_PROVIDER=k8s-1.13.3 # choose this provider
export KUBEVIRT_NUM_NODES=3 # control-plane + two nodes
make cluster-up
```

## Local

Allows provisioning a single-control-plane k8s cluster based on latest upstream k8s
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

First add your provider in the [kubevirtci](https://github.com/kubevirt/kubevirtci) repository.
After this provider is merged, update our copy of this repository with the following command:

```bash
make bump-kubevirtci
```
