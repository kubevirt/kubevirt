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

## Run them anywhere inside of container

```
# Create directory for data
mkdir -p /tmp/kubevirt-tests-data
# Put kubeconfig into data directory
cp ~/.kube/config /tmp/kubevirt-tests-data/kubeconfig
# Run container with data volume
docker run --rm \
    -v /tmp/kubevirt-tests-data:/kubevirt-testing/data:rw,z \
    kubevirt/tests:latest
```
