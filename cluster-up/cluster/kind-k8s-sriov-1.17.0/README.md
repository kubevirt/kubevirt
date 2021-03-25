# K8S 1.17.0 with sriov in a Kind cluster

Provides a pre-deployed k8s cluster with version 1.17.0 that runs using [kind](https://github.com/kubernetes-sigs/kind) The cluster is completely ephemeral and is recreated on every cluster restart. 
The KubeVirt containers are built on the local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

This version also expects to have sriov-enabled nics on the current host, and will move physical interfaces into the `kind`'s cluster worker node(s) so that they can be used through multus.

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=kind-k8s-sriov-1.17.0
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster-up/kubectl.sh get nodes
NAME                  STATUS   ROLES    AGE     VERSION
sriov-control-plane   Ready    master   6m14s   v1.17.0
sriov-worker          Ready    worker   5m36s   v1.17.0
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=kind-k8s-sriov-1.17.0
make cluster-down
```

This destroys the whole cluster. 

## Setting a custom kind version

In order to use a custom kind image / kind version,
export KIND_NODE_IMAGE, KIND_VERSION, KUBECTL_PATH before running cluster-up.
For example in order to use kind 0.9.0 (which is based on k8s-1.19.1) use:
```bash
export KIND_NODE_IMAGE="kindest/node:v1.19.1@sha256:98cf5288864662e37115e362b23e4369c8c4a408f99cbc06e58ac30ddc721600"
export KIND_VERSION="0.9.0"
export KUBECTL_PATH="/usr/bin/kubectl"
```
This allows users to test or use custom images / different kind versions before making them official.
See https://github.com/kubernetes-sigs/kind/releases for details about node images according to the kind version.

## Running multi sriov clusters locally
Kubevirtci sriov provider supports running two clusters side by side with few known limitations.

General considerations:

- A sriov PF must be available for each cluster.
In order to achieve that, there are two options:
1. Assign just one PF for each worker node of each cluster by using `export PF_COUNT_PER_NODE=1` (this is the default value).
2. Optional method: `export PF_BLACKLIST=<PF names>` the non used PFs, in order to prevent them from being allocated to the current cluster.
The user can list the PFs that should not be allocated to the current cluster, keeping in mind
that at least one (or 2 in case of migration), should not be listed, so they would be allocated for the current cluster.
Note: another reason to blacklist a PF, is in case its has a defect or should be kept for other operations (for example sniffing).
- The cluster names must be different.
This can be achieved by setting `export CLUSTER_NAME=sriov2` on the 2nd cluster.
The default `CLUSTER_NAME` is `sriov`.
The 2nd cluster registry would be exposed at `localhost:5001` automatically, once the `CLUSTER_NAME`
is set to a non default value.
- Each cluster should be created on its own git clone folder, i.e
`/root/project/kubevirtci1`
`/root/project/kubevirtci2`
In order to switch between them, change dir to that folder and set the env variables `KUBECONFIG` and `KUBEVIRT_PROVIDER`.
- In case only one PF exists, for example if running on prow which will assign only one PF per job in its own DinD,
Kubevirtci is agnostic and nothing needs to be done, since all conditions above are met.
- Upper limit of the number of clusters that can be run on the same time equals number of PFs / number of PFs per cluster,
therefore, in case there is only one PF, only one cluster can be created.
Locally the actual limit currently supported is two clusters.
- Kubevirtci supports starting `cluster-up` simultaneously, since it is capable of handling race conditions,
when allocating PFs.
- In order to use `make cluster-down` please make sure the right `CLUSTER_NAME` is exported.
