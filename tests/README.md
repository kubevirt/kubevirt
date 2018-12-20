# Integration tests

Integration tests require a running Kubevirt cluster.  Once you have a running
Kubevirt cluster, you can use the `-master` and the `-kubeconfig` flags to
point the tests to the cluster.

## Run them on an arbitrary KubeVirt installation

```
cd tests # from the git repo root folder
go test -kubeconfig=path/to/my/config
```

## Run them on one of the core KubeVirt providers

There is a make target to run this with the config
taken from hack/config.sh:

```
# from the git repo root folder
make functest
```
