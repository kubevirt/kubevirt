# OpenShift 3.10.0 in ephemeral containers

Provides a pre-deployed OpenShift Origin with version 3.10.0 purely in docker
containers with qemu. The provided VMs are completely ephemeral and are
recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are the pushed to a registry which is exposed at
`localhost:5000`.

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=os-3.11.0
export KUBEVIRT_NUM_NODES=2 # master + one nodes
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME      STATUS    ROLES                  AGE       VERSION
node01    Ready     compute,infra,master   22m       v1.10.0+b81c8f8
node02    Ready     compute                19m       v1.10.0+b81c8f8
```

## OpenShift Web Console

If you want to get access to OpenShift web console you will need to add one line to `/etc/hosts`
```bash
echo "127.0.0.1 node01" >> /etc/hosts
```

The background is that the openshift webconsole will always try to redirect to
an authenticator listening at `https://node01:8443`. If this exact url is not
reachable from web-console redirects, then the authentication will always fail.

Use the default user `admin:admin` to log in.

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=os-3.11.0
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
