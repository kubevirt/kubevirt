# Kubernetes 1.9.3 in ephemeral containers

Provides a pre-deployed Kubernetes with version 1.9.3 purely in docker
containers with qemu. The provided VMs are completely ephemeral and are
recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are the pushed to a registry which is exposed at
`localhost:5000`.

## Bringing the cluster up

```bash
export PROVIDER=k8s-1.9.3
export VAGRANT_NUM_NODES=1 # master + one nodes
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME      STATUS     ROLES     AGE       VERSION
node01    NotReady   master    31s       v1.9.3
node02    NotReady   <none>    5s        v1.9.3
```

## Bringing the cluster down

```bash
export PROVIDER=k8s-1.9.3
make cluster-down
```

This destroys the whole cluster. Recreating the cluster is fast, since k8s is
already pre-deployed. The only state which is kept is the state of the local
docker registry.

## Destroying the docker registry state

The docker registry survives a `make cluster-down`. It's state is stored in a
docker volume called `kubevirt_registry`. If the volume gets too big or the
volume contains corrupt data, it can be deleted with

```bash
docker volume rm kubevirt_registry
```
