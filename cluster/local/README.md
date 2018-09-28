# Local Kubernets Provider

This provider allows developing against bleeding-edge Kubernetes code. The
k8s sources will be compiled and a single-node cluster will be started.

## Bringing the cluster up

First get the k8s sources:

```bash
go get -u -d k8s.io/kubernetes
```

Then compile and start the cluster:

```bash
export KUBEVIRT_PROVIDER=local
make cluster-up
```

The cluster can be accessed as usual:

```bash
$ cluster/kubectl.sh get nodes
NAME     STATUS   ROLES    AGE     VERSION
kubdev   Ready    <none>   5m20s   v1.12.0-beta.2
```

Note: you may need to cherry-pick
[acdb1b0e9855ab671f2972f10605d20cad26284b](https://github.com/kubernetes/kubernetes/commit/acdb1b0e9855ab671f2972f10605d20cad26284b)
if it's not present in your kubernetes tree yet.
