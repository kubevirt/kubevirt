# Kubernetes 1.9.3 in vagrant VM

Start vagrant VM and deploy k8s with version 1.9.3 on it.
It will deploy k8s only first time when you start a VM.

## Bringing the cluster up

```bash
export PROVIDER=vagrant-kubernetes
export VAGRANT_NUM_NODES=1
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME      STATUS     ROLES     AGE       VERSION
master    NotReady   master    31s       v1.9.3
node0     NotReady   <none>    5s        v1.9.3
```

## Bringing the cluster down

```bash
export PROVIDER=vagrant-kubernetes
make cluster-down
```

It will shutdown vagrant VM without destroy it.
