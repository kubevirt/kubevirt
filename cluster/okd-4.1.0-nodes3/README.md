# OKD 4.1.0-rc.0 in ephemeral containers

Provides a pre-deployed OKD with version 4.1.0 purely in docker
containers with libvirt. The provided VMs are completely ephemeral and are
recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are the pushed to a registry which is exposed at
`localhost:5000`.

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=okd-4.1.0-nodes3
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ kubectl get nodes
NAME                          STATUS                     ROLES    AGE   VERSION
test-1-r8q8h-master-0         Ready                      master   96m   v1.13.4+cb455d664
test-1-r8q8h-worker-0-k9flc   Ready                      worker   93m   v1.13.4+cb455d664
test-1-r8q8h-worker-0-nknsj   Ready                      worker   93m   v1.13.4+cb455d664
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=okd-4.1.0-nodes3
make cluster-down
```

This destroys the whole cluster. Recreating the cluster is fast, since OKD is
already pre-deployed. The only state which is kept is the state of the local
docker registry.

## Destroying the docker registry state

The docker registry survives a `make cluster-down`. It's state is stored in a
docker volume called `kubevirt_registry`. If the volume gets too big or the
volume contains corrupt data, it can be deleted with

```bash
docker volume rm kubevirt_registry
```
