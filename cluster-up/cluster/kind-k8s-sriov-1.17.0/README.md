# K8S 1.17.0 with sriov in a Kind cluster

Provides a pre-deployed k8s cluster with version 1.17.0 that runs using [kind](https://github.com/kubernetes-sigs/kind) The cluster is completely ephemeral and is recreated on every cluster restart. 
The KubeVirt containers are built on the local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

This version also expects to have sriov-enabed nics on the current host, and will move all the physical interfaces and virtual interfaces into the `kind`'s cluster master node so that they can be used through multus.

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=kind-k8s-sriov-1.17.0
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster-up/kubectl.sh get nodes
NAME                  STATUS    ROLES     AGE       VERSION
sriov-control-plane   Ready     master    2m33s     v1.17.0
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