# K8S 1.17 in a Kind cluster

Provides a pre-deployed k8s cluster with version 1.17 that runs using [kind](https://github.com/kubernetes-sigs/kind) The cluster is completely ephemeral and is recreated on every cluster restart.
The KubeVirt containers are built on the local machine and are then pushed to a registry which is exposed at
`localhost:5000`.


## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=kind-k8s-1.17
export KUBEVIRT_NUM_NODES=2 # master + one node
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster-up/kubectl.sh get nodes
NAME                        STATUS   ROLES    AGE    VERSION
kind-1.17-control-plane   Ready    master   105s   v1.17.x
kind-1.17-worker          Ready    <none>   71s    v1.17.x
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=kind-k8s-1.17
make cluster-down
```

This destroys the whole cluster.

