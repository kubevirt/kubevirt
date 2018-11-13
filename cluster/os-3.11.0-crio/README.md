# OpenShift 3.10.0 with CRI-O in ephemeral containers

Provides a pre-deployed OpenShift Origin with version 3.10.0 with CRI-O support purely in docker
containers with qemu. The provided VMs are completely ephemeral and are
recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are the pushed to a registry which is exposed at
`localhost:5000`.

## Bringing the cluster up

```bash
export PROVIDER=os-3.11.0-crio
export VAGRANT_NUM_NODES=1 # master + one nodes
make cluster-up
```

If you want to get access to OpenShift web console you will need to add line to `/etc/hosts`
```bash
echo "127.0.0.1 node01" >> /etc/hosts
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME      STATUS    ROLES     AGE       VERSION
node01    Ready     master    1h        v1.9.1+a0ce1bc657
node02    Ready     <none>    46s       v1.9.1+a0ce1bc657
```

## Bringing the cluster down

```bash
export PROVIDER=os-3.11.0-crio
make cluster-down
```

This destroys the whole cluster. Recreating the cluster is fast, since OpenShift
is already pre-deployed. The only state which is kept is the state of the local
docker registry.

## Destroying the docker registry state

The docker registry survives a `make cluster-down`. It's state is stored in a
docker volume called `kubevirt_registry`. If the volume gets too big or the
volume contains corrupt data, it can be deleted with

```bash
docker volume rm kubevirt_registry
```
