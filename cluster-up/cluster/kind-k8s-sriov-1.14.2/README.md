# K8S 1.14.2 with sriov in a Kind cluster

Provides a pre-deployed k8s cluster with version 1.14.2 that runs using [kind](https://github.com/kubernetes-sigs/kind) The cluster is completely ephemeral and is recreated on every cluster restart. 
The KubeVirt containers are built on the local machine and are the pushed to a registry which is exposed at
`localhost:5000`.

This version also expects to have sriov-enabed nics on the current host, and will move all the physical interfaces and virtual interfaces into the `kind`'s cluster master node so that they can be used through multus.

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=kind-k8s-sriov-1.14.2
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster-up/kubectl.sh get nodes
NAME                  STATUS    ROLES     AGE       VERSION
sriov-control-plane   Ready     master    2m33s     v1.14.2
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=kind-k8s-sriov-1.14.2
make cluster-down
```

This destroys the whole cluster. 

