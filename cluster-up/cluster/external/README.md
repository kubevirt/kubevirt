# External Kubernetes Provider

This provider works with an existing, provisioned Kubernetes cluster.
An external Docker registry is recommended for serving images.
Unlike with other providers, lifecycles of the cluster and registry are not managed.
The build machine should be a client of the cluster.

## Verifying connectivity

```bash
export KUBEVIRT_PROVIDER=external
export DOCKER_PREFIX=myregistry:5000/kubevirt
export KUBECONFIG=mycluster.conf
export IMAGE_PULL_POLICY=Always
make cluster-up
```


