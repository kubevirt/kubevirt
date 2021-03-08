# Container Image for Functional Tests

This repository ships a container image to run functional tests against installed systems. 
The up-to-date image is pushed to `quay.io/kubevirt/hyperconverged-cluster-functest:<csv-version>-unstable` after every merged change.

## How to run

***Locally***
```shell
 docker run --env KUBECONFIG=/tmp/conf \
  -v $KUBECONFIG:/tmp/conf \
  quay.io/kubevirt/hyperconverged-cluster-functest:1.4.0-unstable 
```

> If the cluster is running locally, you may need to pass `--network host` so that clients in the docker image can access the cluster.

***In cluster***
```shell
kubectl create clusterrolebinding func-cluster-admin --clusterrole=cluster-admin --serviceaccount=kubevirt-hyperconverged:functest
kubectl create serviceaccount functest
kubectl run functest --image=quay.io/kubevirt/hyperconverged-cluster-functest:1.4.0-unstable --serviceaccount=functest
```


## Requirements
- If you are running locally, you have to set `KUBECONFIG` variable and mount kubeconfig into that path.
- If your systems are running in a namespace different from `kubevirt-hypercongerged`, you have to set the environment variable `INSTALLED_NAMESPACE` 


## Arguments

The arguments passed to this container image are directly passed to `func-tests.test` binary in the image.

## Example Command

```shell
docker run  --network=host  \
  --env KUBECONFIG=/tmp/conf \
  -v $KUBECONFIG:/tmp/conf \
  -v /tmp/out:/tmp/out  \
  quay.io/kubevirt/hyperconverged-cluster-functest:1.4.0-unstable  \
  --polarion-execution=true \
  --polarion-project-id=CNV \
  --polarion-report-file=/tmp/out/polarion_results.xml 
  --test-suite-params="env_tier=tier1" \
  --junit-output=/tmp/out/junit.xml \
  --polarion-custom-plannedin=234235

```
