# K8S 1.23.13 with SR-IOV in a Kind cluster

Provides a pre-deployed containerized k8s cluster with version 1.23.13 that runs
using [KinD](https://github.com/kubernetes-sigs/kind)
The cluster is completely ephemeral and is recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

This version also expects to have SR-IOV enabled nics (SR-IOV Physical Function) on the current host, and will move
physical interfaces into the `KinD`'s cluster worker node(s) so that they can be used through multus and SR-IOV
components.

This providers also deploys [multus](https://github.com/k8snetworkplumbingwg/multus-cni)
, [sriov-cni](https://github.com/k8snetworkplumbingwg/sriov-cni)
and [sriov-device-plugin](https://github.com/k8snetworkplumbingwg/sriov-network-device-plugin).

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=kind-1.23-sriov
export KUBEVIRT_NUM_NODES=3
make cluster-up

$ cluster-up/kubectl.sh get nodes
NAME                  STATUS   ROLES                  AGE   VERSION
sriov-control-plane   Ready    control-plane,master   20h   v1.23.13
sriov-worker          Ready    worker                 20h   v1.23.13
sriov-worker2         Ready    worker                 20h   v1.23.13

$ cluster-up/kubectl.sh get pods -n kube-system -l app=multus
NAME                         READY   STATUS    RESTARTS   AGE
kube-multus-ds-amd64-d45n4   1/1     Running   0          20h
kube-multus-ds-amd64-g26xh   1/1     Running   0          20h
kube-multus-ds-amd64-mfh7c   1/1     Running   0          20h

$ cluster-up/kubectl.sh get pods -n sriov -l app=sriov-cni
NAME                            READY   STATUS    RESTARTS   AGE
kube-sriov-cni-ds-amd64-fv5cr   1/1     Running   0          20h
kube-sriov-cni-ds-amd64-q95q9   1/1     Running   0          20h

$ cluster-up/kubectl.sh get pods -n sriov -l app=sriovdp
NAME                                   READY   STATUS    RESTARTS   AGE
kube-sriov-device-plugin-amd64-h7h84   1/1     Running   0          20h
kube-sriov-device-plugin-amd64-xrr5z   1/1     Running   0          20h
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=kind-1.23-sriov
make cluster-down
```

This destroys the whole cluster, and moves the SR-IOV nics to the root network namespace.

## Setting a custom kind version

In order to use a custom kind image / kind version, export `KIND_NODE_IMAGE`, `KIND_VERSION`, `KUBECTL_PATH` before
running cluster-up. For example in order to use kind 0.9.0 (which is based on k8s-1.19.1) use:

```bash
export KIND_NODE_IMAGE="kindest/node:v1.19.1@sha256:98cf5288864662e37115e362b23e4369c8c4a408f99cbc06e58ac30ddc721600"
export KIND_VERSION="0.9.0"
export KUBECTL_PATH="/usr/bin/kubectl"
```

This allows users to test or use custom images / different kind versions before making them official.
See https://github.com/kubernetes-sigs/kind/releases for details about node images according to the kind version.

## Running multi SR-IOV clusters locally

Kubevirtci SR-IOV provider supports running two clusters side by side with few known limitations.

General considerations:

- A SR-IOV PF must be available for each cluster. In order to achieve that, there are two options:

1. Assign just one PF for each worker node of each cluster by using `export PF_COUNT_PER_NODE=1` (this is the default
   value).
2. Optional method: `export PF_BLACKLIST=<PF names>` the non used PFs, in order to prevent them from being allocated to
   the current cluster. The user can list the PFs that should not be allocated to the current cluster, keeping in mind
   that at least one (or 2 in case of migration), should not be listed, so they would be allocated for the current
   cluster. Note: another reason to blacklist a PF, is in case its has a defect or should be kept for other operations (
   for example sniffing).

- Clusters should be created one by another and not in parallel (to avoid races over SR-IOV PF's).
- The cluster names must be different. This can be achieved by setting `export CLUSTER_NAME=sriov2` on the 2nd cluster.
  The default `CLUSTER_NAME` is `sriov`. The 2nd cluster registry would be exposed at `localhost:5001` automatically,
  once the `CLUSTER_NAME`
  is set to a non default value.
- Each cluster should be created on its own git clone folder, i.e:
  `/root/project/kubevirtci1`
  `/root/project/kubevirtci2`
  In order to switch between them, change dir to that folder and set the env variables `KUBECONFIG`
  and `KUBEVIRT_PROVIDER`.
- In case only one PF exists, for example if running on prow which will assign only one PF per job in its own DinD,
  Kubevirtci is agnostic and nothing needs to be done, since all conditions above are met.
- Upper limit of the number of clusters that can be run on the same time equals number of PFs / number of PFs per
  cluster, therefore, in case there is only one PF, only one cluster can be created. Locally the actual limit currently
  supported is two clusters.
- In order to use `make cluster-down` please make sure the right `CLUSTER_NAME` is exported.
