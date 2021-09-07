# Kubernetes 1.x in ephemeral containers

Provides a pre-deployed Kubernetes with version 1.x purely in docker
containers with qemu. The provided VMs are completely ephemeral and are
recreated on every cluster restart.

## Docker registry

There's a docker registry available which is exposed at `localhost:5000`.

## Choosing a cluster version

The env variable `KUBEVIRT_PROVIDER` tells kubevirtci what cluster version to spin up.

```bash
export KUBEVIRT_PROVIDER=k8s-1.21   # choose kubevirtci provider version by subdirectory name
```

## Bringing the cluster up

```bash
export KUBEVIRT_NUM_NODES=2         # master + one node
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME      STATUS     ROLES     AGE       VERSION
node01    NotReady   master    31s       v1.21.1
node02    NotReady   <none>    5s        v1.21.1
```

Note: for further configuration environment variables please see [cluster-up/hack/common.sh](../hack/common.sh)

## Bringing the cluster up with cluster-network-addons-operator provisioned

```bash
export KUBEVIRT_WITH_CNAO=true
make cluster-up
```

To get more info about CNAO you can check the github project documentation
here https://github.com/kubevirt/cluster-network-addons-operator

## Bringing the cluster up with cgroup v2

```bash
export KUBEVIRT_CGROUPV2=true
make cluster-up
```

## Enabling IPv6 connectivity

In order to be able to reach from the cluster to the host's IPv6 network, IPv6
has to be enabled on your Docker. Add following to your
`/etc/docker/daemon.json` and restart docker service:

```json
{
    "ipv6": true,
    "fixed-cidr-v6": "2001:db8:1::/64"
}
```

```bash
systemctl restart docker
```

With an IPv6-connected host, you may want the pods to be able to reach the rest
of the IPv6 world, too. In order to allow that, enable IPv6 NAT on your host:

```bash
ip6tables -t nat -A POSTROUTING -s 2001:db8:1::/64 -j MASQUERADE
```

## Bringing the cluster down

```bash
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
