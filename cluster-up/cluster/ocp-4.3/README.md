# OCP 4.3 in ephemeral containers

Provides a pre-deployed OCP with version 4.3 purely in docker
containers with libvirt. The provided VMs are completely ephemeral and are
recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

## Bringing the cluster up

The container is stored at a private repository at quay.io/kubevirtci, you
have to ask for pull permissions there and do a docker login before cluster-up

```bash
docker login -u [quay user] -p [quay password] quay.io
```

```bash
export KUBEVIRT_PROVIDER=ocp-4.3
export KUBEVIRT_NUM_NODES=3 # master + two workers
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME                          STATUS   ROLES    AGE   VERSION
test-1-82xp6-master-0         Ready    master   62m   v1.12.4+509916ce1
test-1-82xp6-worker-0-wxf27   Ready    worker   57m   v1.12.4+509916ce1
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=ocp-4.3
make cluster-down
```

This destroys the whole cluster. Recreating the cluster is fast, since OCP is
already pre-deployed. The only state which is kept is the state of the local
docker registry.

## Destroying the docker registry state

The docker registry survives a `make cluster-down`. It's state is stored in a
docker volume called `kubevirt_registry`. If the volume gets too big or the
volume contains corrupt data, it can be deleted with

```bash
docker volume rm kubevirt_registry
```
