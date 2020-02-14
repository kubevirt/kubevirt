# K8S 1.14.2 in a Kind cluster

Provides a pre-deployed k8s cluster with version 1.17.0 that runs using [kind](https://github.com/kubernetes-sigs/kind) The cluster is completely ephemeral and is recreated on every cluster restart. 
The KubeVirt containers are built on the local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

cluster is brought up with ipv6 support but without flannel or multi nic support

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=kind-k8s-1.17.0-ipv6
export KUBEVIRT_NUM_NODES=2 # master + one node
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster-up/kubectl.sh get nodes
NAME                        STATUS   ROLES    AGE    VERSION
kind-1.17.0-control-plane   Ready    master   105s   v1.14.2
kind-1.17.0-worker          Ready    <none>   71s    v1.14.2
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=kind-k8s-1.17.0
make cluster-down
```

This destroys the whole cluster. 

