# Test HCO

Provide a simple container which contains `kubectl` and can retrieve and
install HCO.

## How to run it

```
docker run --rm -it -v $PWD/kubeconfig:/kubeconfig -e KUBECONFIG=/kubeconfig kubevirt/hco-tests:latest
```
