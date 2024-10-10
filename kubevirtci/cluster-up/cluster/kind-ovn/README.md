# OVN K8S in a Kind cluster

Provides a k8s cluster that runs using [KinD](https://github.com/kubernetes-sigs/kind)
The cluster is completely ephemeral and is recreated on every cluster restart. The KubeVirt containers are built on the
local machine and are then pushed to a registry which is exposed at
`localhost:5000`.

## Bringing the cluster up

```bash
export KUBEVIRT_PROVIDER=kind-ovn
make cluster-up
```

## Bringing the cluster down

```bash
export KUBEVIRT_PROVIDER=kind-ovn
make cluster-down
```

## FAQ

In case the cluster deployment fails, you need to make sure you have enough watches
add those to /etc/sysctl.conf, and apply it `sysctl -p /etc/sysctl.conf`.
```
sysctl fs.inotify.max_user_watches=1048576
sysctl fs.inotify.max_user_instances=512
```
