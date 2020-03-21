# Kubernetes 1.17 in ephemeral containers

Provides a pre-deployed Kubernetes with version 1.17 purely in docker
containers with qemu. The provided VMs are completely ephemeral and are
recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=k8s-1.17
export KUBEVIRT_NUM_NODES=2 # master + one node
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME      STATUS     ROLES     AGE       VERSION
node01    NotReady   master    31s       v1.17.1
node02    NotReady   <none>    5s        v1.17.1
```

## Bringing the cluster up with cluster-network-addons-operator provisioned

```bash
export KUBEVIRT_PROVIDER=k8s-1.17
export KUBEVIRT_NUM_NODES=2 # master + one node
export KUBEVIRT_WITH_CNAO=true
make cluster-up
```

To get more info about CNAO you can check the github project documentation
here https://github.com/kubevirt/cluster-network-addons-operator

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=k8s-1.17
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
